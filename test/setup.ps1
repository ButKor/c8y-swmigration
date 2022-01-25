param (
    [string]$prefix,
    [int]$ctSw,
    [int]$ctVersionPerSw

)

1..$ctSw | ForEach-Object {
    $swName = "sw_$($_)"
    1..$ctVersionPerSw | ForEach-Object {
        $versionName = "v$($_)"

        @{
            c8y_Global = @{}
            name       = "$($prefix)_$($swName)"
            type       = "c8y_Software"
            url        = "https://example.org/$($swName)/$($versionName).deb"
            version    = "$($prefix)_$($versionName)"
        } | ConvertTo-Json -Depth 5 > swpackage.json
        c8y inventory create --template ./swpackage.json --force -o json -c --select "**"
    } 
}
