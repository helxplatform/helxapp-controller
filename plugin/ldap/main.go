package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/go-ldap/ldap/v3"
	"github.com/gorilla/mux"
)

// LDAPConfig holds the configuration for the LDAP server
type LDAPConfig struct {
	Host     string
	Port     int
	BindDN   string
	Password string
	BaseDN   string
}

// User represents the user profile information
type User struct {
	UID                string   `json:"uid"`
	CommonName         string   `json:"commonName"`
	Surname            string   `json:"surname"`
	GivenName          string   `json:"givenName"`
	DisplayName        string   `json:"displayName"` // Include displayName
	Email              string   `json:"email"`
	Telephone          string   `json:"telephoneNumber"`
	Organization       string   `json:"organization"`
	OrganizationalUnit string   `json:"organizationalUnit"`
	RunAsUser          string   `json:"runAsUser,omitempty"`
	RunAsGroup         string   `json:"runAsGroup,omitempty"`
	FsGroup            string   `json:"fsGroup,omitempty"`
	SupplementalGroups []string `json:"supplementalGroups,omitempty"`
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
	return nil
}

// searchLDAP searches the LDAP server for a user by UID
func searchLDAP(username string) (*User, error) {
	// Connect to LDAP
	l, err := ldap.Dial("tcp", fmt.Sprintf("%s:%d", ldapConfig.Host, ldapConfig.Port))
	if err != nil {
		return nil, err
	}
	defer l.Close()

	// Bind with credentials
	err = l.Bind(ldapConfig.BindDN, ldapConfig.Password)
	if err != nil {
		return nil, err
	}

	// Search for the given username
	searchRequest := ldap.NewSearchRequest(
		ldapConfig.BaseDN,
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		fmt.Sprintf("(uid=%s)", ldap.EscapeFilter(username)),
		[]string{"uid", "cn", "sn", "givenName", "displayName", "mail", "telephoneNumber", "o", "ou", "runAsUser", "runAsGroup", "fsGroup", "supplementalGroups"},
		nil,
	)

	sr, err := l.Search(searchRequest)
	if err != nil {
		return nil, err
	}

	if len(sr.Entries) == 0 {
		return nil, fmt.Errorf("user not found")
	}

	entry := sr.Entries[0]
	user := &User{
		UID:                entry.GetAttributeValue("uid"),
		CommonName:         entry.GetAttributeValue("cn"),
		Surname:            entry.GetAttributeValue("sn"),
		GivenName:          entry.GetAttributeValue("givenName"),
		DisplayName:        entry.GetAttributeValue("displayName"), // Retrieve and set displayName
		Email:              entry.GetAttributeValue("mail"),
		Telephone:          entry.GetAttributeValue("telephoneNumber"),
		Organization:       entry.GetAttributeValue("o"),
		OrganizationalUnit: entry.GetAttributeValue("ou"),
		RunAsUser:          entry.GetAttributeValue("runAsUser"),
		RunAsGroup:         entry.GetAttributeValue("runAsGroup"),
		FsGroup:            entry.GetAttributeValue("fsGroup"),
		SupplementalGroups: entry.GetAttributeValues("supplementalGroups"),
	}

	return user, nil
}

// userHandler handles the /users/{username} route
func userHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	username := vars["username"]

	user, err := searchLDAP(username)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
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

func main() {
	configPath := "/etc/config/ldap-config.json"
	err := loadConfig(configPath)

	if err != nil {
		log.Fatalf("Error loading config: %s", err)
	}

	r := mux.NewRouter()
	r.HandleFunc("/users/{username}", userHandler)
	log.Fatal(http.ListenAndServe(":8180", r))
}
