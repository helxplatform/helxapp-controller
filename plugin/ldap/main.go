package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"reflect"
	"sort"
	"strconv"

	"github.com/go-ldap/ldap/v3"
	"github.com/gorilla/mux"
)

// LDAPConfig holds the configuration for the LDAP server
type LDAPConfig struct {
	Host           string `json:"host"`
	Port           int    `json:"port"`
	Username       string `json:"username"`
	PasswordFile   string `json:"passwordFile"`
	UserBaseDN     string `json:"user_base_dn"`
	GroupBaseDN    string `json:"group_base_dn"`
	// Legacy fields for backward compatibility
	BindDN   string `json:"bindDN,omitempty"`
	Password string `json:"password,omitempty"`
	BaseDN   string `json:"baseDN,omitempty"`
}

// PosixGroup represents an LDAP posixGroup entry
type PosixGroup struct {
	CN         string   `json:"cn"`
	GIDNumber  string   `json:"gidNumber"`
	MemberUIDs []string `json:"memberUids,omitempty"`
}

// User represents the user profile information
type User struct {
	UID                string       `json:"uid"`
	CommonName         string       `json:"commonName"`
	Surname            string       `json:"surname"`
	GivenName          string       `json:"givenName"`
	DisplayName        string       `json:"displayName"`
	Email              string       `json:"email"`
	Telephone          string       `json:"telephoneNumber"`
	Organization       string       `json:"organization"`
	OrganizationalUnit string       `json:"organizationalUnit"`
	RunAsUser          string       `json:"runAsUser,omitempty"`
	RunAsGroup         string       `json:"runAsGroup,omitempty"`
	FsGroup            string       `json:"fsGroup,omitempty"`
	SupplementalGroups []string     `json:"supplementalGroups,omitempty"`
	Groups             []string     `json:"groups,omitempty"`
	PosixGroups        []PosixGroup `json:"posixGroups,omitempty"`
	UserAlias          string       `json:"userAlias,omitempty"`

	// posixAccount fields
	UIDNumber     string `json:"uidNumber,omitempty"`
	GIDNumber     string `json:"gidNumber,omitempty"`
	HomeDirectory string `json:"homeDirectory,omitempty"`
	LoginShell    string `json:"loginShell,omitempty"`
}

var ldapConfig LDAPConfig

func loadConfig(path string) error {
	file, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	err = json.Unmarshal(file, &ldapConfig)
	if err != nil {
		return err
	}

	// Load password from file if specified
	if ldapConfig.PasswordFile != "" {
		pw, err := os.ReadFile(ldapConfig.PasswordFile)
		if err != nil {
			return fmt.Errorf("failed to read password file %s: %w", ldapConfig.PasswordFile, err)
		}
		ldapConfig.Password = string(pw)
	}

	// Apply legacy field fallbacks
	if ldapConfig.Username == "" && ldapConfig.BindDN != "" {
		ldapConfig.Username = ldapConfig.BindDN
	}
	if ldapConfig.UserBaseDN == "" && ldapConfig.BaseDN != "" {
		ldapConfig.UserBaseDN = ldapConfig.BaseDN
	}
	if ldapConfig.Port == 0 {
		ldapConfig.Port = 389
	}
	return nil
}

// MergeEmpty fills empty string fields in dst with non-empty values from src.
func MergeEmpty[T any](dst, src *T) {
	dv := reflect.ValueOf(dst).Elem()
	sv := reflect.ValueOf(src).Elem()

	for i := 0; i < dv.NumField(); i++ {
		dstField := dv.Field(i)
		srcField := sv.Field(i)

		if dstField.Kind() == reflect.String && dstField.String() == "" {
			dstField.Set(srcField)
		}
	}
}

func extractCN(dn string) string {
	parts := splitDN(dn)
	for _, part := range parts {
		trimmed := trimSpace(part)
		if len(trimmed) > 3 && trimmed[:3] == "cn=" {
			return trimmed[3:]
		}
	}
	return ""
}

func splitDN(s string) []string {
	return splitString(s, ',')
}

func splitString(s string, sep byte) []string {
	var parts []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == sep {
			parts = append(parts, s[start:i])
			start = i + 1
		}
	}
	parts = append(parts, s[start:])
	return parts
}

func trimSpace(s string) string {
	start, end := 0, len(s)
	for start < end && s[start] == ' ' {
		start++
	}
	for end > start && s[end-1] == ' ' {
		end--
	}
	return s[start:end]
}

func extractGroupNames(groupDNs []string) []string {
	groupCNs := make([]string, len(groupDNs))
	for i, dn := range groupDNs {
		groupCNs[i] = extractCN(dn)
	}
	return groupCNs
}

// constructSupplementalGroups merges supplementalGroups attr values
// with posixGroup GID numbers into a deduplicated sorted list.
func constructSupplementalGroups(user *User) []string {
	seen := make(map[string]bool)
	var result []string

	for _, sg := range user.SupplementalGroups {
		if !seen[sg] {
			seen[sg] = true
			result = append(result, sg)
		}
	}
	for _, pg := range user.PosixGroups {
		if pg.GIDNumber != "" && !seen[pg.GIDNumber] {
			seen[pg.GIDNumber] = true
			result = append(result, pg.GIDNumber)
		}
	}

	sort.Slice(result, func(i, j int) bool {
		ni, _ := strconv.Atoi(result[i])
		nj, _ := strconv.Atoi(result[j])
		return ni < nj
	})
	return result
}

// searchLDAP searches the LDAP server for a user by UID
func searchLDAP(username string) (*User, error) {
	bindUsername := ldapConfig.Username
	if bindUsername == "" {
		bindUsername = ldapConfig.BindDN
	}
	userBaseDN := ldapConfig.UserBaseDN
	if userBaseDN == "" {
		userBaseDN = ldapConfig.BaseDN
	}

	// Connect to LDAP
	l, err := ldap.Dial("tcp", fmt.Sprintf("%s:%d", ldapConfig.Host, ldapConfig.Port))
	if err != nil {
		return nil, err
	}
	defer l.Close()

	// Bind with credentials
	err = l.Bind(bindUsername, ldapConfig.Password)
	if err != nil {
		return nil, err
	}

	// Search for the given username
	searchRequest := ldap.NewSearchRequest(
		userBaseDN,
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		fmt.Sprintf("(uid=%s)", ldap.EscapeFilter(username)),
		[]string{
			"uid", "cn", "sn", "givenName", "displayName",
			"mail", "telephoneNumber", "o", "ou",
			"runAsUser", "runAsGroup", "fsGroup", "supplementalGroups",
			"memberOf", "userAlias",
			// posixAccount attributes
			"uidNumber", "gidNumber", "homeDirectory", "loginShell",
		},
		nil,
	)

	sr, err := l.Search(searchRequest)
	if err != nil {
		return nil, err
	}

	if len(sr.Entries) == 0 {
		slog.Info("LDAP user not found", "username", username)
		return nil, nil
	}

	entry := sr.Entries[0]
	user := &User{
		UID:                entry.GetAttributeValue("uid"),
		CommonName:         entry.GetAttributeValue("cn"),
		Surname:            entry.GetAttributeValue("sn"),
		GivenName:          entry.GetAttributeValue("givenName"),
		DisplayName:        entry.GetAttributeValue("displayName"),
		Email:              entry.GetAttributeValue("mail"),
		Telephone:          entry.GetAttributeValue("telephoneNumber"),
		Organization:       entry.GetAttributeValue("o"),
		OrganizationalUnit: entry.GetAttributeValue("ou"),
		RunAsUser:          entry.GetAttributeValue("runAsUser"),
		RunAsGroup:         entry.GetAttributeValue("runAsGroup"),
		FsGroup:            entry.GetAttributeValue("fsGroup"),
		SupplementalGroups: entry.GetAttributeValues("supplementalGroups"),
		Groups:             extractGroupNames(entry.GetAttributeValues("memberOf")),
		UserAlias:          entry.GetAttributeValue("userAlias"),
		UIDNumber:          entry.GetAttributeValue("uidNumber"),
		GIDNumber:          entry.GetAttributeValue("gidNumber"),
		HomeDirectory:      entry.GetAttributeValue("homeDirectory"),
		LoginShell:         entry.GetAttributeValue("loginShell"),
	}

	// Fetch posixGroups the user belongs to
	groupBaseDN := ldapConfig.GroupBaseDN
	if groupBaseDN == "" {
		groupBaseDN = userBaseDN
	}
	posixGroupSearchRequest := ldap.NewSearchRequest(
		groupBaseDN,
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		fmt.Sprintf("(&(objectClass=posixGroup)(memberUid=%s))", ldap.EscapeFilter(user.UID)),
		[]string{"cn", "gidNumber", "memberUid"},
		nil,
	)

	posixGroupResult, err := l.Search(posixGroupSearchRequest)
	if err != nil {
		slog.Error("failed to search posixGroups", "uid", user.UID, "err", err)
	} else {
		posixGroups := make([]PosixGroup, len(posixGroupResult.Entries))
		for i, entry := range posixGroupResult.Entries {
			posixGroups[i] = PosixGroup{
				CN:         entry.GetAttributeValue("cn"),
				GIDNumber:  entry.GetAttributeValue("gidNumber"),
				MemberUIDs: entry.GetAttributeValues("memberUid"),
			}
		}
		user.PosixGroups = posixGroups
	}

	// Resolve UserAlias: fetch alias user and merge empty fields
	if user.UserAlias != "" {
		slog.Info("found alias for user", "uid", user.UID, "alias", user.UserAlias)
		if aliasUser, _ := searchLDAP(user.UserAlias); aliasUser != nil {
			slog.Info("merging alias user", "uid", user.UID, "alias", user.UserAlias)
			MergeEmpty(user, aliasUser)
			if uidNumber, err := strconv.Atoi(user.UIDNumber); err == nil && uidNumber == -1 {
				user.UIDNumber = aliasUser.UIDNumber
			}
			if gidNumber, err := strconv.Atoi(user.GIDNumber); err == nil && gidNumber == -1 {
				user.GIDNumber = aliasUser.GIDNumber
			}
		}
	}

	// Apply RunAsUser/RunAsGroup fallbacks from posixAccount fields
	if user.RunAsUser == "" && user.UIDNumber != "" {
		user.RunAsUser = user.UIDNumber
	}
	if user.RunAsGroup == "" && user.GIDNumber != "" {
		user.RunAsGroup = user.GIDNumber
	}

	// Merge supplementalGroups with posixGroup GIDs
	user.SupplementalGroups = constructSupplementalGroups(user)

	slog.Info("retrieved LDAP user", "user", user.UID)

	return user, nil
}

// userHandler handles the /users/{username} route
func userHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	username := vars["username"]

	user, err := searchLDAP(username)
	if err != nil {
		slog.Error("LDAP search failed", "username", username, "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if user == nil {
		http.Error(w, "user not found", http.StatusNotFound)
		return
	}

	jsonResponse, err := json.Marshal(user)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonResponse)
}

// LivenessProbeHandler checks if the application is running
func LivenessProbeHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("alive"))
}

// ReadinessProbeHandler checks if the application is ready to accept traffic
func ReadinessProbeHandler(w http.ResponseWriter, r *http.Request) {
	err := loadConfig("/etc/config/config.json")
	if err != nil {
		http.Error(w, "Not Ready", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ready"))
}

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	configPath := "/etc/config/config.json"
	err := loadConfig(configPath)

	if err != nil {
		slog.Error("failed to load config", "err", err)
		os.Exit(1)
	}

	slog.Info("LDAP plugin starting",
		"host", ldapConfig.Host,
		"port", ldapConfig.Port,
		"userBaseDN", ldapConfig.UserBaseDN,
		"groupBaseDN", ldapConfig.GroupBaseDN,
	)

	r := mux.NewRouter()
	r.HandleFunc("/users/{username}", userHandler)
	r.HandleFunc("/healthz", LivenessProbeHandler)
	r.HandleFunc("/readyz", ReadinessProbeHandler)

	slog.Info("listening on :8080")
	if err := http.ListenAndServe(":8080", r); err != nil {
		slog.Error("server failed", "err", err)
		os.Exit(1)
	}
}
