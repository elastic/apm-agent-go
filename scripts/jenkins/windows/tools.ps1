# Installs golang on Windows.
#
# # Run script:
# .\install-go.ps1 -version 1.5.3
#
# # Download and run script:
# $env:GOVERSION = '1.5.3'
# iex ((new-object net.webclient).DownloadString('SCRIPT_URL_HERE'))
#
# # Further references:
# https://gist.github.com/andrewkroh/2c93f8a5953f6093a505
Param(
    [String]$goroot,
    [String]$version,
    [switch]$h = $false,
    [switch]$help = $false
)

$SCRIPT=$MyInvocation.MyCommand.Name

function print_usage() {
  echo @"
Download and install Golang on Windows. It sets the GOROOT environment
variable and adds GOROOT\bin to the PATH environment variable.
Usage:
  $SCRIPT -goroot C:\go -version 1.5.3
Options:
  -h | -help
    Print the help menu.
  -goroot
    Golang root path where to install to. Required.
  -version
    Golang version to install. Required. Env variable GOROOT
"@
}

if ($help -or $h) {
  print_usage
  exit 0
}
if ($version -eq "") {
    $version = $env:GOVERSION
}
if ($version -eq "" ) {
  Write-Error "Error: -version is required"
  print_usage
  exit 1
}
if ($goroot -eq "" ) {
  Write-Error "Error: -goroot is required"
  print_usage
  exit 1
}

$downloadDir = $env:TEMP
$packageName = 'golang'
$url32 = 'https://storage.googleapis.com/golang/go' + $version + '.windows-386.zip'
$url64 = 'https://storage.googleapis.com/golang/go' + $version + '.windows-amd64.zip'
$goroot = "$goroot$version"

# Determine type of system
if ($ENV:PROCESSOR_ARCHITECTURE -eq "AMD64") {
  $url = $url64
} else {
  $url = $url32
}

if (Test-Path "$goroot\bin\go") {
  Write-Host "Go is installed to $goroot"
  exit
}

echo "Downloading $url"
$zip = "$downloadDir\golang-$version.zip"
if (!(Test-Path "$zip")) {
  $downloader = new-object System.Net.WebClient
  $downloader.DownloadFile($url, $zip)
}

echo "Extracting $zip to $goroot"
if (Test-Path "$downloadDir\go") {
  rm -Force -Recurse -Path "$downloadDir\go"
}
Add-Type -AssemblyName System.IO.Compression.FileSystem
[System.IO.Compression.ZipFile]::ExtractToDirectory("$zip", $downloadDir)
mv "$downloadDir\go" $goroot
