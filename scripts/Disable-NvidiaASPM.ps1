$base = 'HKLM:\SYSTEM\CurrentControlSet\Control\Class\{4d36e968-e325-11ce-bfc1-08002be10318}'

Get-ChildItem $base -ErrorAction SilentlyContinue | ForEach-Object {
    $p = Get-ItemProperty $_.PSPath -ErrorAction SilentlyContinue
    if ($p.ProviderName -match 'NVIDIA' -or $p.DriverDesc -match 'NVIDIA|CMP') {
        Set-ItemProperty -Path $_.PSPath -Name 'DisableAspm' -Value 1 -Type DWord -Force -ErrorAction SilentlyContinue
        Set-ItemProperty -Path $_.PSPath -Name 'RMDisableLinkDownshift' -Value 1 -Type DWord -Force -ErrorAction SilentlyContinue
    }
}
