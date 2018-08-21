#!/usr/bin/env bash

shopt -s extglob
set -o errtrace
set -o errexit

LOOM_HOME=/opt/loom
PROTOBUF_VERSION=3.5.1

log()  { printf "%b\n" "$*"; }
debug(){ [[ ${loom_debug_flag:-0} -eq 0 ]] || printf "%b\n" "$*" >&2; }
fail() { log "\nERROR: $*\n" >&2 ; exit 1 ; }

loom_install_initialize()
{
  if [ ! -d ${LOOM_HOME} ] || [ ! -d ${LOOM_HOME}/bin ] || [ ! -d ${LOOM_HOME}/workdir ]; then
    log "Creating directory ${LOOM_HOME} ${LOOM_HOME}/bin ${LOOM_HOME}/workdir"
    sudo mkdir -p ${LOOM_HOME} ${LOOM_HOME}/bin ${LOOM_HOME}/workdir
    sudo chown -R ${USER} ${LOOM_HOME}
  else
    log "${LOOM_HOME} ${LOOM_HOME}/bin ${LOOM_HOME}/workdir already exists."
  fi
}

loom_detect_os()
{
  log "Detecting operating system..."
  if grep -i Microsoft /proc/version >/dev/null 2>&1; then
    platform=linux
    os_type=windows
  elif grep -i ubuntu /proc/version >/dev/null 2>&1; then
    platform=linux
    os_type=ubuntu
  elif uname | grep -i darwin >/dev/null 2>&1; then
    platform=osx
    os_type=darwin
  else
    fail "Unable to detect OS..."
  fi
  log "Found ${platform} on ${os_type}."
}

loom_install_commands_setup()
{
  brew_installed=false

  \which which >/dev/null 2>&1 || fail "Could not find 'which' command, make sure it's available first before continuing installation."
  \which grep >/dev/null 2>&1 || fail "Could not find 'grep' command, make sure it's available first before continuing installation."
  \which unzip >/dev/null 2>&1 || fail "Could not find 'unzip' command, make sure it's available first before continuing installation."

  if \which curlx >/dev/null 2>&1; then
    download_command="curl -sL -o"
  elif \which wget >/dev/null 2>&1; then
    download_command="wget -q -O"
  fi

  if \which brew >/dev/null 2>&1; then
    brew_installed=true
  fi
}

loom_install_dependencies()
{
  protobuf_satisfied=false
  if \which protoc >/dev/null 2>&1; then
    PROTOBUF_VERSION_INSTALLED=$(protoc --version | awk '{print $2}')
    log "Detected protobuf version ${PROTOBUF_VERSION_INSTALLED}"
    if [ "${PROTOBUF_VERSION_INSTALLED}" != "${PROTOBUF_VERSION}" ]; then
      log "Protobuf version ${PROTOBUF_VERSION_INSTALLED} does not match required version ${PROTOBUF_VERSION}"
    else
      protobuf_satisfied=true
    fi
  fi

  if ! \which protoc >/dev/null 2>&1 || ! $protobuf_satisfied; then
    log "Installing protobuf version ${PROTOBUF_VERSION}..."
    if $brew_installed; then
      brew install protobuf
    else
	    $download_command /tmp/protoc-${PROTOBUF_VERSION}-${platform}-x86_64.zip \
        https://github.com/google/protobuf/releases/download/v${PROTOBUF_VERSION}/protoc-${PROTOBUF_VERSION}-${platform}-x86_64.zip
	    sudo unzip /tmp/protoc-${PROTOBUF_VERSION}-${platform}-x86_64.zip -d /usr/local
      sudo chmod 755 /usr/local/bin/protoc
	    sudo find /usr/local/include/google -type d -exec chmod 755 -- {} +
      sudo find /usr/local/include/google -type f -exec chmod 644 -- {} +
	    rm /tmp/protoc-${PROTOBUF_VERSION}-${platform}-x86_64.zip
    fi
  fi

  if ! \which protoc >/dev/null 2>&1; then
    fail "protobuf installation failed"
  fi
}

loom_download_executable()
{
  if $brew_installed; then
    log "Installing loom using homebrew..."
    brew tap loomnetwork/client
    brew install loom
  else
    echo "Downloading loom executable..."
    $download_command ${LOOM_HOME}/bin/loom https://private.delegatecall.com/loom/${platform}/latest/loom
    chmod +x ${LOOM_HOME}/bin/loom
  fi
  if [ ! -h "/usr/local/bin/loom" ]; then
    sudo rm -f "/usr/local/bin/loom"
    sudo ln -s ${LOOM_HOME}/bin/loom /usr/local/bin/loom
  fi
}

loom_configure()
{
  cd ${LOOM_HOME}/workdir
  log "Running loom init in ${LOOM_HOME}/workdir"
  loom init

  log "Creating genesis.json in ${LOOM_HOME}/workdir"
  cat > ${LOOM_HOME}/workdir/genesis.json <<EOF
{
    "contracts": [
    ]
}
EOF

  log "Creating loom.yml in ${LOOM_HOME}/workdir"
  echo 'QueryServerHost: "tcp://0.0.0.0:9999"' > ${LOOM_HOME}/workdir/loom.yml
}

loom_create_startup()
{
  loom_startup=""
  if [ "$os_type" = "windows" ] || [ "$os_type" = "darwin" ]; then
    log "Startup script for ${platform} on ${os_type} is currently unsupported"
    return
  fi

  if \which systemctl >/dev/null 2>&1; then
    log "Creating systemd startup script"
    cat > /tmp/loom.service <<EOF
[Unit]
Description=Loom
After=network.target

[Service]
Type=simple
User=${USER}
WorkingDirectory=${LOOM_HOME}/workdir
ExecStart=/usr/local/bin/loom run
Restart=always
RestartSec=2
StartLimitInterval=0
LimitNOFILE=500000
StandardOutput=syslog
StandardError=syslog

[Install]
WantedBy=multi-user.target
EOF
    sudo mv /tmp/loom.service /etc/systemd/system/loom.service
    sudo systemctl daemon-reload
    sudo systemctl start loom.service
    loom_startup=systemd
  else
    log "Startup script for ${platform} on ${os_type} is currently available only for systemd"
  fi
}

loom_done()
{
  if [ "$loom_startup" = "systemd" ]; then
    printf "%b" "
Startup script has been installed via systemd.

To view its status:

  \$ sudo systemctl status loom.service

To enable it at startup:

  \$ sudo systemctl enable loom.service

To view logs:

  \$ sudo journalctl -u loom.service -f
"
  else
    printf "%b" "
To run loom:

  \$ cd ${LOOM_HOME}/workdir
  \$ loom run
"
  fi
}

loom_install()
{
  loom_install_initialize
  loom_detect_os
  loom_install_commands_setup
  loom_install_dependencies
  loom_download_executable
  loom_configure
  loom_create_startup
  loom_done
}

loom_install "$@"
