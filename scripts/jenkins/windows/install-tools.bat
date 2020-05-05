::
:: This script install the required tools
::

if "%GO_VERSION%" == "1.14.2" (
    git clone https://github.com/v1v/chocolatey-packages .chocolatey-packages
    cd .chocolatey-packages\golang\1.14.2
    choco pack
    choco install golang -y --no-progress -s .
) else (
    choco config set cacheLocation %WORKSPACE%
    choco install golang --version %GO_VERSION% -y --no-progress
)
