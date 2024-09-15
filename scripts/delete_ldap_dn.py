#!/usr/bin/env python

import ldap
import argparse

def delete_ldap_user(dn, ldap_config):
    try:
        # Open a connection to the LDAP server
        l = ldap.initialize(ldap_config['ldap_server'])
        l.protocol_version = ldap.VERSION3
        l.simple_bind_s(ldap_config['bind_dn'], ldap_config['bind_password'])

        # Attempt to delete the DN
        l.delete_s(dn)
        print(f"Successfully deleted DN: {dn}")
    except ldap.NO_SUCH_OBJECT:
        print(f"No such entry: {dn}")
    except ldap.LDAPError as e:
        print(f"Error deleting DN {dn}: {e}")
    finally:
        l.unbind_s()

def main():
    parser = argparse.ArgumentParser(description='Delete a single LDAP entry specified by DN.')
    parser.add_argument('dn', help='The DN of the LDAP entry to delete')
    parser.add_argument('--ldap-server', required=True, help='LDAP server URL, e.g., ldap://localhost')
    parser.add_argument('--bind-dn', default='cn=admin,dc=example,dc=org', help='Bind DN for LDAP authentication')
    parser.add_argument('--bind-password', required=True, help='Password for Bind DN')

    args = parser.parse_args()

    ldap_config = {
        'ldap_server': args.ldap_server,
        'bind_dn': args.bind_dn,
        'bind_password': args.bind_password,
    }

    delete_ldap_user(args.dn, ldap_config)

if __name__ == "__main__":
    main()
