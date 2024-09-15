#!/usr/bin/env python

import ldap
import ldap.modlist as modlist
import yaml
import argparse

def create_ldap_user(user, ldap_config):
    try:
        # Open a connection to the LDAP server
        l = ldap.initialize(ldap_config['ldap_server'])
        l.protocol_version = ldap.VERSION3
        l.simple_bind_s(ldap_config['bind_dn'], ldap_config['bind_password'])

        # DN of the new user
        dn = f"uid={user['uid']},{ldap_config['base_dn']}"

        # Build the user attributes dictionary
        attrs = {
            'objectClass': [b'inetOrgPerson', b'organizationalPerson', b'person', b'kubernetesSC', b'top'],
            'uid': [user['uid'].encode('utf-8')],
            'cn': [user['cn'].encode('utf-8')],
            'sn': [user['sn'].encode('utf-8')],
            'mail': [user.get('email', '').encode('utf-8')],
            'telephoneNumber': [user.get('telephoneNumber', '').encode('utf-8')],
            'o': [user.get('o', '').encode('utf-8')],
            'ou': [user.get('ou', '').encode('utf-8')],
            'givenName': [user.get('givenName', '').encode('utf-8')],
            'displayName': [user.get('displayName', '').encode('utf-8')],
            'supplementalGroups': [str(group).encode('utf-8') for group in user.get('supplementalGroups', [])],
            'runAsUser': [str(user['runAsUser']).encode('utf-8')],
            'runAsGroup': [str(user['runAsGroup']).encode('utf-8')],
            'fsGroup': [str(user['fsGroup']).encode('utf-8')]
        }

        # Check if the user already exists
        try:
            existing_attrs = l.search_s(dn, ldap.SCOPE_BASE)[0][1]
            if existing_attrs:
                # User exists, prepare to update
                ldif = modlist.modifyModlist(existing_attrs, attrs)
                if ldif:  # Only perform the update if there are changes
                    l.modify_s(dn, ldif)
                    print(f"User {user['uid']} updated successfully.")
                else:
                    print(f"No updates necessary for user {user['uid']}.")
                return
        except ldap.NO_SUCH_OBJECT:
            pass  # No existing user, proceed to create

        # Create the user if not exists
        ldif = modlist.addModlist(attrs)
        l.add_s(dn, ldif)
        print(f"User {user['uid']} created successfully.")
    except ldap.LDAPError as e:
        print(f"Error in creating user {user['uid']}:", e)
    finally:
        l.unbind_s()

def load_users_from_yaml(path):
    with open(path, 'r') as file:
        return yaml.safe_load(file)

def main():
    parser = argparse.ArgumentParser(description='Create LDAP users from a YAML file.')
    parser.add_argument('yaml_file', help='Path to YAML file with user data')
    parser.add_argument('--ldap-server', required=True, help='LDAP server URL, e.g., ldap://localhost')
    parser.add_argument('--bind-dn', default='cn=admin,dc=example,dc=org', help='Bind DN for LDAP authentication')
    parser.add_argument('--bind-password', required=True, help='Password for Bind DN')
    parser.add_argument('--base-dn', default='ou=users,dc=example,dc=org', help='Base DN where the users will be created')

    args = parser.parse_args()

    ldap_config = {
        'ldap_server': args.ldap_server,
        'bind_dn': args.bind_dn,
        'bind_password': args.bind_password,
        'base_dn': args.base_dn,
    }

    users = load_users_from_yaml(args.yaml_file)
    for user in users['users']:
        create_ldap_user(user, ldap_config)

if __name__ == "__main__":
    main()
