#!/bin/bash

# ref: https://stackoverflow.com/a/7069755/244009
# ref: https://jonalmeida.com/posts/2013/05/26/different-ways-to-implement-flags-in-bash/
# ref: http://tldp.org/LDP/abs/html/comparison-ops.html

export STASH_NAMESPACE=kube-system
export STASH_SERVICE_ACCOUNT=default
export STASH_ENABLE_RBAC=false
export STASH_RUN_ON_MASTER=0
export STASH_ROLE_TYPE=ClusterRole

show_help() {
    echo "stash.sh - install stash operator"
    echo " "
    echo "stash.sh [options]"
    echo " "
    echo "options:"
    echo "-h, --help                         show brief help"
    echo "-n, --namespace=NAMESPACE          specify namespace (default: kube-system)"
    echo "    --rbac                         create RBAC roles and bindings"
    echo "    --run-on-master                run stash operator on master"
}

while test $# -gt 0; do
    case "$1" in
        -h|--help)
            show_help
            exit 0
            ;;
        -n)
            shift
            if test $# -gt 0; then
                export STASH_NAMESPACE=$1
            else
                echo "no namespace specified"
                exit 1
            fi
            shift
            ;;
        --namespace*)
            export STASH_NAMESPACE=`echo $1 | sed -e 's/^[^=]*=//g'`
            shift
            ;;
        --rbac)
            export STASH_SERVICE_ACCOUNT=stash-operator
            export STASH_ENABLE_RBAC=true
            shift
            ;;
        --run-on-master)
            export STASH_RUN_ON_MASTER=1
            shift
            ;;
        *)
            show_help
            exit 1
            ;;
    esac
done

env | sort | grep STASH*
echo ""

curl -fsSL https://raw.githubusercontent.com/appscode/stash/0.6.3/hack/deploy/operator.yaml | envsubst | kubectl apply -f -

if [ "$STASH_ENABLE_RBAC" = true ]; then
    curl -fsSL https://raw.githubusercontent.com/appscode/stash/0.6.3/hack/deploy/rbac.yaml | envsubst | kubectl apply -f -
fi

if [ "$STASH_RUN_ON_MASTER" -eq 1 ]; then
    kubectl patch deploy stash-operator -n $STASH_NAMESPACE \
      --patch="$(curl -fsSL https://raw.githubusercontent.com/appscode/stash/0.6.3/hack/deploy/run-on-master.yaml)"
fi
