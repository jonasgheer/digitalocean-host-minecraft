package main

const (
	helpText = `
Usage: mc COMMAND

Commands:
  start
  stop
  download

Run 'mc COMMAND -help' for more information on a command`

	startupScript = `#!/bin/bash
cd /root
apt update -y
apt install openjdk-8-jre-headless -y
wget https://launcher.mojang.com/v1/objects/3dc3d84a581f14691199cf6831b71ed1296a9fdf/server.jar -O minecraft.jar
java -Xms512M -Xmx1024M -jar minecraft.jar nogui
sed -i -e s/eula=false/eula=true/g eula.txt
screen -dmS minecraft java -Xmx1024M -Xms512M -jar minecraft.jar nogui`
)
