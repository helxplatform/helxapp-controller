#!/usr/bin/env python

import ldap
import argparse
import yaml

def fetch_user_details(ldap_server, bind_dn, bind_password, search_base):
    try:
        l = ldap.initialize(ldap_server)
        l.protocol_version = ldap.VERSION3
        l.simple_bind_s(bind_dn, bind_password)

        # Update the list to include kubernetesSC attributes
        search_scope = ldap.SCOPE_SUBTREE
        search_filter = '(objectClass=inetOrgPerson)'
        retrieve_attributes = [
            'uid', 'cn', 'sn', 'mail', 'telephoneNumber',
            'givenName', 'displayName', 'o', 'ou',
            'runAsUser', 'runAsGroup', 'fsGroup', 'supplementalGroups'  # kubernetesSC attributes
        ]
        ldap_result_id = l.search(search_base, search_scope, search_filter, retrieve_attributes)

        result_set = []
        while True:
            result_type, result_data = l.result(ldap_result_id, 0)
            if result_data == []:
                break
            if result_type == ldap.RES_SEARCH_ENTRY:
                entry = result_data[0][1]  # Get the dictionary of attributes
                processed_entry = {}
                for attr in retrieve_attributes:
                    if attr in ['runAsUser', 'runAsGroup', 'fsGroup'] and entry.get(attr):
                        processed_entry[attr] = int(entry[attr][0].decode('utf-8'))
                    elif attr == 'supplementalGroups' and entry.get(attr):
                        processed_entry[attr] = [int(x.decode('utf-8')) for x in entry[attr]]
                    else:
                        processed_entry[attr] = entry.get(attr, [b''])[0].decode('utf-8') if entry.get(attr) else ''
                result_set.append(processed_entry)

        l.unbind_s()
        return result_set
    except ldap.LDAPError as e:
        print("LDAP error:", e)
        return []
    finally:
        try:
            l.unbind_s()
        except:
            pass

def main():
    parser = argparse.ArgumentParser(description='Retrieve and display details for all users in the LDAP directory.')
    parser.add_argument('--ldap-server', required=True, help='LDAP server URL, e.g., ldap://localhost')
    parser.add_argument('--bind-dn', default='cn=admin,dc=example,dc=org', help='Bind DN for LDAP authentication')
    parser.add_argument('--bind-password', required=True, help='Password for Bind DN')
    parser.add_argument('--search-base', default='ou=users,dc=example,dc=org', help='Base DN where the search starts')
    parser.add_argument('--output-format', choices=['text', 'yaml'], default='text', help='Output format of the user data')

    args = parser.parse_args()

    users = fetch_user_details(args.ldap_server, args.bind_dn, args.bind_password, args.search_base)
    if args.output_format == 'yaml':
        print(yaml.dump({'users': users}, default_flow_style=False))
    else:
        for user in users:
            print("User Details:")
            for key, value in user.items():
                print(f"{key}: {value}")
            print("-" * 40)

if __name__ == "__main__":
    main()
