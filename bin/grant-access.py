#!/usr/bin/env python3
"""
Grant a service account the permissions needed to install the helxapp-controller
helm chart in namespace mode (cluster=false).

This must be run by a cluster admin. It creates:
  1. A ClusterRole with full helx.renci.org CRD permissions
  2. A RoleBinding in the target namespace granting those permissions to the SA

The SA can then run: helm install helxapp-controller <chart>

Usage:
  python3 grant-access.py <namespace>:<service-account>

Example:
  python3 grant-access.py jeffw:jeffw
"""

import json
import subprocess
import sys

CLUSTERROLE_NAME = "helxapp-controller-namespace-installer"

CLUSTERROLE = {
    "apiVersion": "rbac.authorization.k8s.io/v1",
    "kind": "ClusterRole",
    "metadata": {"name": CLUSTERROLE_NAME},
    "rules": [
        {
            "apiGroups": ["helx.renci.org"],
            "resources": [
                "helxapps", "helxapps/status", "helxapps/finalizers",
                "helxinsts", "helxinsts/status", "helxinsts/finalizers",
                "helxusers", "helxusers/status", "helxusers/finalizers",
            ],
            "verbs": ["get", "list", "watch", "create", "update", "patch", "delete"],
        }
    ],
}


def rolebinding(namespace, sa):
    return {
        "apiVersion": "rbac.authorization.k8s.io/v1",
        "kind": "RoleBinding",
        "metadata": {"name": CLUSTERROLE_NAME, "namespace": namespace},
        "roleRef": {
            "apiGroup": "rbac.authorization.k8s.io",
            "kind": "ClusterRole",
            "name": CLUSTERROLE_NAME,
        },
        "subjects": [
            {"kind": "ServiceAccount", "name": sa, "namespace": namespace}
        ],
    }


def kubectl_apply(resource):
    subprocess.run(
        ["kubectl", "apply", "-f", "-"],
        input=json.dumps(resource),
        text=True,
        check=True,
    )


def main():
    if len(sys.argv) != 2 or ":" not in sys.argv[1]:
        print(f"Usage: {sys.argv[0]} <namespace>:<service-account>", file=sys.stderr)
        sys.exit(1)

    namespace, sa = sys.argv[1].split(":", 1)

    print(f"Creating ClusterRole {CLUSTERROLE_NAME}...")
    kubectl_apply(CLUSTERROLE)

    print(f"Creating RoleBinding in namespace {namespace} for SA {sa}...")
    kubectl_apply(rolebinding(namespace, sa))

    print(f"Done. SA {namespace}:{sa} can now install the helm chart in namespace {namespace}.")


if __name__ == "__main__":
    main()
