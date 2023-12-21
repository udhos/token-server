# Usage

[Helm](https://helm.sh) must be installed to use the charts.  Please refer to
Helm's [documentation](https://helm.sh/docs) to get started.

Once Helm has been set up correctly, add the repo as follows:

    helm repo add token-server https://udhos.github.io/token-server

Update files from repo:

    helm repo update

Search token-server:

    $ helm search repo token-server -l --version ">=0.0.0"
    NAME                 	    CHART VERSION	APP VERSION	DESCRIPTION
    token-server/token-server	0.1.0        	0.0.0      	A Helm chart for token-server

To install the charts:

    helm install my-token-server token-server/token-server
    #            ^               ^            ^
    #            |               |             \__________ chart
    #            |               |
    #            |                \_______________________ repo
    #            |
    #             \_______________________________________ release (chart instance installed in cluster)

To uninstall the charts:

    helm uninstall my-token-server

# Source

<https://github.com/udhos/token-server>
