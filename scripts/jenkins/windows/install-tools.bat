::
:: This script install the required tools
::

choco config set cacheLocation %WORKSPACE%
:: Install latest as long as https://github.com/chocolatey/chocolatey.org/issues/803
:: is not resolved.
:: choco install golang --version %GO_VERSION% -y --no-progress
choco install golang -y --no-progress
