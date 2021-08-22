#!/bin/sh

## Get Latest Build's URL from master
URL=$(curl -s https://api.github.com/repos/0chain/zboxcli/releases/latest \
| grep "browser_download_url.*zip" \
| cut -d : -f 2,3 \
| tr -d \")

# Download, extract the zip to get the exe and delete the zip
curl -LO ${URL}
unzip -o zbox-windows.zip && rm zbox-windows.zip