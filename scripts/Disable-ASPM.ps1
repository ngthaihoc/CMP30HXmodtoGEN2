$schemes = powercfg -list | ForEach-Object {
    if ($_ -match '([a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12})') {
        $matches[1]
    }
}

foreach ($s in $schemes) {
    powercfg -setacvalueindex $s SUB_PCIEXPRESS ASPM 0 2>$null
    powercfg -setdcvalueindex $s SUB_PCIEXPRESS ASPM 0 2>$null
    powercfg -setacvalueindex $s SUB_SLEEP HYBRIDSLEEP 0 2>$null
    powercfg -setdcvalueindex $s SUB_SLEEP HYBRIDSLEEP 0 2>$null
}

powercfg -setactive SCHEME_CURRENT 2>$null
