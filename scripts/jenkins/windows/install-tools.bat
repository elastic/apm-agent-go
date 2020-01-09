::
:: This script install the required tools
::

choco config set cacheLocation %WORKSPACE%
choco install golang --version %GO_VERSION% -y --no-progress
