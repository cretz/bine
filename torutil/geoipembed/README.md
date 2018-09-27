**How to regen**

With [go-bindata](https://github.com/go-bindata/go-bindata) installed and assuming `tor-static` is present:

    go-bindata -pkg geoipembed -prefix ..\..\..\tor-static\tor\src\config ..\..\..\tor-static\tor\src\config\geoip ..\..\..\tor-static\tor\src\config\geoip6

Then delete the public API, delete the unused imports, remove the generated comments at the package level, and put the
mod time in for `LastUpdated` in `geoipembed`. One day this might all be automated, e.g. download maxmind db ourselves,
gen code, update last updated, etc.