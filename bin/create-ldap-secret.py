#!/usr/bin/env python3
"""
Create a Kubernetes Secret containing the LDAP bind password
for the helxapp-controller LDAP plugin.

Usage:
  python3 create-ldap-secret.py <password> [--name <secret-name>] [--namespace <ns>]

Arguments:
  password                  The LDAP bind password

Options:
  --name, -n <name>         Secret name (default: helxapp-controller-ldap-pw)
  --namespace, -ns <ns>     Target namespace (default: current context namespace)

Examples:
  python3 create-ldap-secret.py 's3cret'
  python3 create-ldap-secret.py 's3cret' --name my-ldap-creds --namespace prod
"""

import argparse
import sys

from kubernetes import client, config
from kubernetes.client.rest import ApiException


DEFAULT_SECRET_NAME = "helxapp-controller-ldap-pw"


def get_current_namespace():
    """Return the namespace from the current kubeconfig context."""
    _, active_context = config.list_kube_config_contexts()
    return active_context.get("context", {}).get("namespace", "default")


def create_or_update_secret(namespace, name, password, force=False):
    v1 = client.CoreV1Api()

    secret = client.V1Secret(
        metadata=client.V1ObjectMeta(name=name, namespace=namespace),
        type="Opaque",
        string_data={"password": password},
    )

    try:
        v1.read_namespaced_secret(name, namespace)
        if not force:
            print(f"Secret {namespace}/{name} already exists. Use --force to overwrite.", file=sys.stderr)
            sys.exit(1)
        v1.replace_namespaced_secret(name, namespace, secret)
        print(f"Secret {namespace}/{name} updated.")
    except ApiException as e:
        if e.status == 404:
            v1.create_namespaced_secret(namespace, secret)
            print(f"Secret {namespace}/{name} created.")
        else:
            raise


def main():
    parser = argparse.ArgumentParser(
        description="Create the LDAP bind-password Secret for the helxapp-controller LDAP plugin."
    )
    parser.add_argument("password", help="LDAP bind password")
    parser.add_argument(
        "--name", "-n", default=DEFAULT_SECRET_NAME, help=f"Secret name (default: {DEFAULT_SECRET_NAME})"
    )
    parser.add_argument(
        "--namespace", "-ns", default=None, help="Target namespace (default: current context namespace)"
    )
    parser.add_argument(
        "--force", "-f", action="store_true", help="Overwrite the secret if it already exists"
    )
    args = parser.parse_args()

    config.load_kube_config()

    namespace = args.namespace or get_current_namespace()

    create_or_update_secret(namespace, args.name, args.password, args.force)
    print(f"Use --set ldapPlugin.passwordSecret={args.name} when installing the chart.")


if __name__ == "__main__":
    main()
