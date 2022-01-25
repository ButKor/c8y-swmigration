ARG VARIANT="1.17-bullseye"
FROM mcr.microsoft.com/vscode/devcontainers/go:0-${VARIANT}

# c8ycli
RUN sudo apt-get install -y curl gnupg2 apt-transport-https
RUN curl https://reubenmiller.github.io/go-c8y-cli-repo/debian/PUBLIC.KEY | gpg --dearmor | sudo tee /usr/share/keyrings/go-c8y-cli-archive-keyring.gpg >/dev/null
RUN sudo sh -c "echo 'deb [signed-by=/usr/share/keyrings/go-c8y-cli-archive-keyring.gpg] http://reubenmiller.github.io/go-c8y-cli-repo/debian stable main' >> /etc/apt/sources.list"
RUN sudo apt-get update
RUN sudo apt-get install go-c8y-cli
# CMD bash -c "/etc/go-c8y-cli/shell/c8y.plugin.sh"

# jsonnet
RUN sudo apt-get update
RUN sudo apt-get -y install jsonnet

# jq
RUN sudo apt install jq -y

# pwsh
RUN sudo wget https://github.com/PowerShell/PowerShell/releases/download/v7.2.1/powershell-lts_7.2.1-1.deb_amd64.deb
RUN sudo apt install ./powershell-lts_7.2.1-1.deb_amd64.deb