#!/usr/bin/env python

import ldap
import argparse

def fetch_all_dns(ldap_server, bind_dn, bind_password, search_base):
    try:
        # Initialize and bind to the LDAP server
        l = ldap.initialize(ldap_server)
        l.protocol_version = ldap.VERSION3
        l.simple_bind_s(bind_dn, bind_password)

        # Define the search scope (subtree to search everything beneath base DN)
        search_scope = ldap.SCOPE_SUBTREE

        # Define the search filter; (objectClass=*) will match all entries
        search_filter = "(objectClass=*)"

        # Perform the search
        ldap_result_id = l.search(search_base, search_scope, search_filter, None)
        result_set = []
        while True:
            result_type, result_data = l.result(ldap_result_id, 0)
            if result_data == []:
                break
            if result_type == ldap.RES_SEARCH_ENTRY:
                result_set.append(result_data[0][0])  # result_data[0][0] is the DN
        return result_set
    except ldap.LDAPError as e:
        print("LDAP error:", e)
        return []  # Return an empty list on failure
    finally:
        l.unbind_s()

def main():
    parser = argparse.ArgumentParser(description='Retrieve all Distinguished Names (DNs) from an LDAP server.')
    parser.add_argument('--ldap-server', required=True, help='LDAP server URL, e.g., ldap://localhost')
    parser.add_argument('--bind-dn', default='cn=admin,dc=example,dc=org', help='Bind DN for LDAP authentication')
    parser.add_argument('--bind-password', required=True, help='Password for Bind DN')
    parser.add_argument('--search-base', default='dc=example,dc=org', help='Base DN where the search starts')

    args = parser.parse_args()

    # Retrieve all DNs from the LDAP server
    dns = fetch_all_dns(args.ldap_server, args.bind_dn, args.bind_password, args.search_base)
    for dn in dns:
        print(dn)

if __name__ == "__main__":
    main()
