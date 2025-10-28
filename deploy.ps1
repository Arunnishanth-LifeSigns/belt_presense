# deploy.ps1 (Refactored for Installer Package method)

[CmdletBinding()]
param ()

# --- Configuration ---
# MODIFICATION: The BastionHost IP has been updated.
$BastionHost = "13.127.85.4"
$TargetHost = "10.15.133.150"
$LocalPemKeyPath = 'C:\Users\Lifesigns_T14\Downloads\ls-central-dev-common 2.pem'
$BastionPemKeyPath = "~/ls-central-dev-common.pem"
$ServiceName = "belt-presense"
$SshUser = "ubuntu"
# --- End Configuration ---

Start-Transcript -Path ".\deployment-log.txt" -Append
$ErrorActionPreference = 'Stop'

try {
    Write-Host "Starting package-based deployment to $TargetHost via Bastion $BastionHost..." -ForegroundColor Yellow
    
    # =================================================================
    ## 1. Build and Package the Application
    # =================================================================
    Write-Host "Building and packaging the application..." -ForegroundColor Cyan
    
    # Enable CGO and specify the Linux cross-compiler
    $env:CGO_ENABLED = 1
    $env:GOOS = "linux"
    $env:GOARCH = "amd64"
    $env:CC = "x86_64-w64-mingw32-gcc"

    $BinaryName = "$ServiceName-svc"
    # The -tags flag is crucial to link librdkafka statically
    go build -tags musl -ldflags "-extldflags -static" -o ./$BinaryName ./cmd/main.go

    # Create a temporary package directory
    $PackageDir = ".\package"
    if (Test-Path $PackageDir) { Remove-Item -Recurse -Force $PackageDir }
    New-Item -ItemType Directory -Path $PackageDir | Out-Null
    
    # Copy all necessary files into the package directory
    Copy-Item -Path ./$BinaryName -Destination $PackageDir
    # MODIFICATION: Copy .env.prod and rename it to .env for the package.
    Copy-Item -Path .\.env.prod -Destination "$PackageDir\.env"
    Copy-Item -Path .\install.sh -Destination $PackageDir
    Copy-Item -Path .\uninstall.sh -Destination $PackageDir

    # Create a compressed tarball (package.tar.gz)
    $PackageName = "package.tar.gz"
    tar -czvf $PackageName -C $PackageDir .
    
    Write-Host "Package created: $PackageName" -ForegroundColor Green
    Write-Host "----------------------------------------------------"

    # =================================================================
    ## 2. Transfer and Execute the Package
    # =================================================================
    $TempRemotePath = "/tmp"
    $RemotePackagePath = "$($TempRemotePath)/$($PackageName)"

    # A. Copy package from Local -> Bastion
    Write-Host "--> (Step 1/3) Copying package to Bastion host..."
    scp -i $LocalPemKeyPath $PackageName "$($SshUser)@$($BastionHost):$TempRemotePath/"

    # B. Move package from Bastion -> Target
    Write-Host "--> (Step 2/3) Moving package to Target host..."
    $scpCommand = "scp -i $BastionPemKeyPath -o StrictHostKeyChecking=no $RemotePackagePath $($SshUser)@$($TargetHost):~/"
    ssh -i $LocalPemKeyPath "$($SshUser)@$($BastionHost)" $scpCommand

    # C. Unpack and run the installer on Target
    Write-Host "--> (Step 3/3) Running installer on Target host..."
    $installCommand = @"
set -e
echo '--- Unpacking and installing on target ---'
INSTALL_DIR="~/install_package"
mkdir -p \$INSTALL_DIR
tar -xzvf ~/$PackageName -C \$INSTALL_DIR
sudo \$INSTALL_DIR/install.sh
echo '--- Cleaning up installation files ---'
rm ~/$PackageName
rm -rf \$INSTALL_DIR
"@
    $remoteExecCommand = "ssh -i $BastionPemKeyPath -o StrictHostKeyChecking=no $($SshUser)@$($TargetHost) '$($installCommand)'"
    ssh -i $LocalPemKeyPath "$($SshUser)@$($BastionHost)" $remoteExecCommand

    Write-Host "Deployment to $TargetHost completed successfully!" -ForegroundColor Green
}
catch { Write-Host "An error occurred: $($_.Exception.Message)" -ForegroundColor Red }
finally { Stop-Transcript }