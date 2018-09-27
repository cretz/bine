**How to regen**

With [go-bindata](https://github.com/go-bindata/go-bindata) installed and assuming `tor-static` is present:

    go-bindata -pkg geoipembed -prefix ..\..\..\tor-static\tor\src\config ..\..\..\tor-static\tor\src\config\geoip ..\..\..\tor-static\tor\src\config\geoip6

Then just go delete the public API and unused imports. Then just put the mod time in for `LastUpdated` in `geoipembed`.
One day this might all be automated, e.g. download maxmind db ourselves, gen code, update last updated, etc.