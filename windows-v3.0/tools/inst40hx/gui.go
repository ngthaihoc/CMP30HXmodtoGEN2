package main

// v2.6.0: τ▓╛τ«ÇσìòτòîΘ¥ó GUI (walk) ΓÇö µùáΘÇëΘí╣σìíπÇüµùáτè╢µÇüΦí¿µá╝, σ«ëΦúàσÖ¿σÅ¬τ«íσ«ëΦúàπÇé
// µëôσ╝Çµù╢Φç¬σè¿σÅ¬Φ»╗µë½µÅÅΣ╕Çµ¼í(Σ╕ìσ▒òτñ║µÿÄτ╗å): τ╗ôµ₧£σÅ¬τö¿Σ║Ä
//   Γæá τ╗äΣ╗╢σ«ëΦúàσî║τÜä"τÄ»σóâµÅÉτñ║"Φíî(µ£¬µúÇµ╡ïσê░σìí/SecureBoot/Legacy τ¡ëΦ¡ªσæè)
//   Γæí τ╝║σñ▒/µ£¬Φ╛╛µáçτ╗äΣ╗╢τÜäΦç¬σè¿Θóäσï╛(σ╖▓ΦúàΣ╕ìσï╛ = Σ╕ìΦªåτ¢û)
// τòîΘ¥óΦç¬Σ╕èΦÇîΣ╕ï:
//   Γæá τ╗äΣ╗╢σ«ëΦúà ΓÇö [σ«ëΦúàµëÇΘÇëτ╗äΣ╗╢] / [Σ╕ÇΘö«σ«îµò┤σ«ëΦúà(σà¿µ╡üτ¿ï)], Φúàσ«îΦç¬σè¿Θçìµë½µ¢┤µû░µÅÉτñ║
//   Γæí Gen2 τ¡ûτòÑ ΓÇö Θ⌐▒σè¿Φ┐ÉΦíîτ¡ûτòÑ + Φç¬σè¿ Stage2 σ¢₧ΘÇÇ + σñ▒Φ┤ÑΘçìΦ»òµ¼íµò░/Θù┤ΘÜö,[Σ┐¥σ¡ÿτ¡ûτòÑ]
//   Γæó µôìΣ╜£µùÑσ┐ù ΓÇö AttachLogSink σ«₧µù╢Φ╛ôσç║(GUI Σ╕Ä CLI σà▒τö¿σà¿Θâ¿σ«₧τÄ░)
// σì╕Φ╜╜Σ╕ÄΦ»ªτ╗åΦ»èµû¡Σ╕ìσ£¿µ£¼τòîΘ¥ó: 40HXUninstaller.exe / -uninstall / 40HXCheck.exeπÇé
// τ║┐τ¿ïτ║ªσ«Ü: OnClicked(UI τ║┐τ¿ï)σÅ¬Φ»╗µÄºΣ╗╢ ΓåÆ goroutine µëºΦíî ΓåÆ UI σÅÿµ¢┤Σ╕Çσ╛ïτ╗Å sync()πÇé

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"40hxcore"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
	"golang.org/x/sys/windows/registry"
)

// ---- τÄ»σóâµë½µÅÅ(σÅ¬Φ»╗; τ╗ôµ₧£σûé tip Σ╕ÄΘóäσï╛ΘÇë, Σ╕ìσ▒òτñ║µÿÄτ╗åΦí¿) ----

type statusItem struct {
	Name string `json:"name"`
	Ok   bool   `json:"ok"`
	Note string `json:"note"`
}

func scanStatus() []statusItem {
	items := []statusItem{}
	legacy := hxcore.FirmwareIsLegacy()
	items = append(items, statusItem{"Chß║┐ ─æß╗Ö Boot", !legacy,
		map[bool]string{true: "UEFI (OK)", false: "Legacy+MBR ΓÇö Cß║ºn chuyß╗ân sang GPT bß║▒ng mbr2gpt"}[legacy]})
	sbOn := hxcore.SecureBootOn()
	items = append(items, statusItem{"Secure Boot", !sbOn,
		map[bool]string{true: "─Éang Bß║¡t (Cß║ºn Tß║»t trong BIOS!)", false: "─É├ú Tß║»t (OK)"}[sbOn]})
	gpuOK := hxcore.FindGPU()
	items = append(items, statusItem{"Card ─æß╗ô hoß║í", gpuOK,
		map[bool]string{true: "─É├ú ph├ít hiß╗çn GPU t╞░╞íng th├¡ch", false: "Ch╞░a ph├ít hiß╗çn ΓÇö Kiß╗âm tra khe cß║»m & driver"}[gpuOK]})
	gspOK := hxcore.GspEnabled()
	items = append(items, statusItem{"GSP (EnableGpuFirmware)", gspOK,
		map[bool]string{true: "─É├ú bß║¡t (OK)", false: "Ch╞░a bß║¡t ΓÇö C├│ thß╗â bß╗ï Code 43 sau mß╗ƒ kho├í"}[gspOK]})

	espEFI := false
	if esp := hxcore.MountESP(); esp != "" {
		if _, err := os.Stat(esp + ":\\EFI\\40HX\\40HXUNLK.EFI"); err == nil {
			espEFI = true
		}
		hxcore.UnmountESP(esp)
	}
	items = append(items, statusItem{"ESP EFI Mß╗ƒ kho├í", espEFI,
		map[bool]string{true: "\\EFI\\40HX\\40HXUNLK.EFI ─æ├ú nß║íp", false: "Ch╞░a nß║íp (Legacy kh├┤ng hß╗ù trß╗ú)"}[espEFI]})
	// σÉ»σè¿Θí╣Σ╕ëµÇü: ΘªûΣ╜ì / σ¡ÿσ£¿Σ╜åΣ╕ìσ£¿ΘªûΣ╜ì / µ£¬σê¢σ╗║
	bootOK := false
	bootNote := "Ch╞░a tß║ío"
	if ex, first, ord := verifyBootEntry(); ex {
		if first {
			bootOK = true
			bootNote = "Tß╗ôn tß║íi v├á nß║▒m ─æß║ºu ti├¬n (displayorder)"
		} else {
			bootNote = "Tß╗ôn tß║íi nh╞░ng ch╞░a ─æß║╖t ─æß║ºu ti├¬n (thß╗⌐ tß╗▒: " + ord + ") ΓÇö Cß║ºn v├áo BIOS chß╗ënh"
		}
	}
	items = append(items, statusItem{"Mß╗Ñc khß╗ƒi ─æß╗Öng BIOS", bootOK, bootNote})

	taskOK, taskStatus, taskResult := hxcore.TaskInfo(gen2TaskName)
	taskNote := "Ch╞░a ─æ─âng k├╜ ΓÇö Sß║╜ kh├┤ng tß╗▒ mß╗ƒ kho├í khi bß║¡t m├íy"
	if taskOK {
		taskNote = "Trß║íng th├íi: " + taskStatus + ", Kß║┐t quß║ú gß║ºn nhß║Ñt: " + taskResult
	}
	items = append(items, statusItem{"T├íc vß╗Ñ tß╗▒ khß╗ƒi ─æß╗Öng", taskOK, taskNote})
	rkOK := runKeyPresent()
	items = append(items, statusItem{"Kho├í Run dß╗▒ ph├▓ng Registry", rkOK,
		map[bool]string{true: "─É├ú ghi (40HXGen2)", false: "Ch╞░a ghi"}[rkOK]})

	deps := hxcore.InspectGen2Drivers()
	if !hxcore.Gen2DriversDeployedOnce() {
		items = append(items, statusItem{"Driver PCIe (Ch╞░a c├ái)", false,
			"Ch╞░a c├│ driver ΓÇö H├úy t├¡ch chß╗ìn [C├ái ─æß║╖t Driver PCIe] b├¬n d╞░ß╗¢i"})
	} else {
		var notes []string
		curOK := true
		cleanEnd := true
		for _, d := range deps {
			svcS := "Dß╗ïch vß╗Ñ ch╞░a ─æ─âng k├╜"
			if d.SvcReg {
				svcS = "Dß╗ïch vß╗Ñ: " + d.SvcStart
				if d.SvcStart == "DISABLED" {
					svcS += " ΓÜáBß╗ï v├┤ hiß╗çu ho├í (c├ái lß║íi ─æß╗â sß╗¡a)"
				}
				if d.SvcRunning {
					svcS += "/─Éang chß║íy"
				}
			}
			notes = append(notes, d.File+": System32="+d.SysState.String()+", "+svcS)
			if d.SysState != hxcore.DrvOk || !d.SvcReg || d.SvcStart == "DISABLED" {
				curOK = false
			}
			if d.SvcReg || d.SysState != hxcore.DrvAbsent {
				cleanEnd = false
			}
		}
		if cleanEnd && hxcore.DriverStrategy() != hxcore.DriverStrategyResident {
			items = append(items, statusItem{"Driver PCIe (─É├ú dß╗ìn dß║╣p sß║ích)", true,
				"─É├ú c├ái; Tß╗▒ dß╗ìn dß║╣p sau khi chß║íy (b├¼nh th╞░ß╗¥ng, lß║ºn ─æ─âng nhß║¡p sau sß║╜ tß╗▒ nß║íp lß║íi)"})
		} else {
			items = append(items, statusItem{"Trß║íng th├íi Driver PCIe", curOK, strings.Join(notes, " | ")})
		}
	}
	if exOK, err := hxcore.DefenderExclusionsPresent(); err != nil {
		items = append(items, statusItem{"Loß║íi trß╗½ Windows Defender", false,
			"Kiß╗âm tra thß║Ñt bß║íi (" + err.Error() + ") ΓÇö Chß║íy lß║íi vß╗¢i quyß╗ün Admin"})
	} else {
		items = append(items, statusItem{"Loß║íi trß╗½ Windows Defender", exOK,
			map[bool]string{true: "─É├ú th├¬m loß║íi trß╗½ cho 2 file .sys v├á th╞░ mß╗Ñc ProgramData", false: "Ch╞░a th├¬m loß║íi trß╗½ ΓÇö Antivirus c├│ thß╗â chß║╖n driver"}[exOK]})
	}

	fsOn := hxcore.FastStartupOn()
	items = append(items, statusItem{"Khß╗ƒi ─æß╗Öng nhanh (Fast Startup)", !fsOn,
		map[bool]string{true: "─Éang Bß║¡t (Khuy├¬n tß║»t ─æß╗â nß║íp UEFI chuß║⌐n)", false: "─É├ú Tß║»t (OK)"}[fsOn]})
	if ac, dc, aspmOK := hxcore.ASPMSavings(); aspmOK {
		off := ac == 0 && dc == 0
		items = append(items, statusItem{"Tiß║┐t kiß╗çm ─æiß╗çn PCIe (ASPM)", off,
			map[bool]string{true: "─É├ú Tß║»t (OK)", false: fmt.Sprintf("─Éang Bß║¡t (AC=%d DC=%d) ΓÇö C├│ thß╗â bß╗ï tß╗Ñt tß╗æc ─æß╗Ö khi nghß╗ë", ac, dc)}[off]})
	}
	return items
}

func runKeyPresent() bool {
	k, err := registry.OpenKey(registry.CURRENT_USER,
		`Software\Microsoft\Windows\CurrentVersion\Run`, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer k.Close()
	_, _, err = k.GetStringValue("40HXGen2")
	return err == nil
}

// ---- µùÑσ┐ùΘ¥óµ¥┐σåÖσàÑσÖ¿ (AttachLogSink τ¢«µáç) ----

type guiLog struct {
	mw *walk.MainWindow
	te *walk.TextEdit
}

func (g *guiLog) Write(p []byte) (int, error) {
	s := string(p)
	if g.mw != nil && g.te != nil {
		// EDIT µÄºΣ╗╢µìóΦíîΘ£ÇΦªü CRLF: τ╗ƒΣ╕Çµèè \n ΦºäΦîâµêÉ \r\n, σÉªσêÖµùÑσ┐ùΣ╝ÜµîñµêÉΣ╕Çµ«╡
		s = strings.ReplaceAll(s, "\r\n", "\n")
		s = strings.ReplaceAll(s, "\r", "\n")
		s = strings.ReplaceAll(s, "\n", "\r\n")
		g.mw.Synchronize(func() { g.te.AppendText(s) })
	}
	return len(p), nil
}

// ---- GUI τè╢µÇü ----

type guiState struct {
	mw       *walk.MainWindow
	log      *guiLog
	teLog    *walk.TextEdit
	tip      *walk.Label
	busyBy   string // σ╜ôσëìσìáτö¿Σ║ÆµûÑτÜäµôìΣ╜£σÉì(""=τ⌐║Θù▓)πÇéσÅ¬σ£¿ UI τ║┐τ¿ïΦ»╗σåÖ:
	// begin() σ£¿ OnClicked(UIτ║┐τ¿ï) Φ░âτö¿, end() τ╗Å sync σ¢₧σê░ UI τ║┐τ¿ï ΓåÆ µùáτ║┐τ¿ïτ½₧Σ║ëπÇé
	lastScan  []statusItem
	lastGuide string // Σ╕èµ¼íµëôσì░τÜäτÄ»σóâµîçσ╝ò(σÅÿσîûµëìµëôσì░, Θÿ▓Θçìµë½σê╖σ▒Å)
	lastAV      string // Σ╕èµ¼íΦ»åσê½σê░τÜäτ¼¼Σ╕ëµû╣µ¥ÇΦ╜»(σÉîΣ╕è)
	lastDefWarn string // "µùá Defender µ¿íσ¥ù"µÅÉτñ║σÄ╗Θçì

	ckGsp, ckDrv, ckEfi, ckTask            *walk.CheckBox
	ckFast, ckAspm, ckPerf, ckDefOff       *walk.CheckBox
	pbInstall, pbFull                      *walk.PushButton

	rbStrategy     [3]*walk.RadioButton
	ckAutoHard     *walk.CheckBox
	neRetryCnt     *walk.NumberEdit
	neRetryMin     *walk.NumberEdit
	pbSave, pbGen2, pbGen2Install *walk.PushButton
}

func (st *guiState) sync(f func()) {
	if st.mw != nil {
		st.mw.Synchronize(f)
	} else {
		f()
	}
}

// begin: σ£¿ UI τ║┐τ¿ï(OnClicked)σÉîµ¡Ñµèóσà¿σ▒ÇΣ║ÆµûÑ ΓÇö Σ╕Çµ¼íσÅ¬σàüΦ«╕Σ╕ÇΣ╕¬Θò┐µôìΣ╜£σ£¿Φ╖æπÇé
// µèóσê░σì│σìáΣ╜Å(busyBy)σ╣╢σÉîµ¡Ñτªüτö¿µîëΘÆ«, µ¥£τ╗¥"Φ┐₧τé╣/σ┐½ΘÇƒσÅîσç╗"σ£¿ goroutine ΘçîµèóΘöüτÜäτ½₧µÇüπÇé
func (st *guiState) begin(what string) bool {
	if st.busyBy != "" {
		fmt.Println("[!] ─Éang thß╗▒c hiß╗çn " + st.busyBy + " ΓÇö " + what + " ─æ├ú bß╗Å qua, vui l├▓ng ─æß╗úi ho├án tß║Ñt rß╗ôi thß╗¡ lß║íi")
		return false
	}
	st.busyBy = what
	return true
}

func (st *guiState) end() {
	st.sync(func() { st.busyBy = "" })
}

// setActionsEnabled: σê¥σºïΦç¬σè¿µë½µÅÅµ£ƒΘù┤τªüτö¿µëºΦíîµîëΘÆ«, Θÿ▓µ¡óΣ╕Äµëïσè¿µôìΣ╜£σ╣╢σÅæµèó IOπÇé
func (st *guiState) setActionsEnabled(on bool) {
	st.sync(func() {
		for _, b := range []*walk.PushButton{st.pbInstall, st.pbFull, st.pbSave, st.pbGen2} {
			if b != nil {
				b.SetEnabled(on)
			}
		}
	})
}

// summaryText: µèè"Σ╕ìσñäτÉåσ░▒ΦúàΣ╕ìΣ╕è/ΦúàΣ║åΣ╣ƒΣ╕ìτöƒµòê"τÜäµ£Çσà│Θö«τè╢µÇüσÉêµêÉσ«ëΦúàσî║Θí╢Θâ¿µÅÉτñ║πÇé
func (st *guiState) summaryText(items []statusItem) string {
	m := map[string]statusItem{}
	for _, it := range items {
		m[it.Name] = it
	}
	if it, ok := m["Card ─æß╗ô hoß║í"]; ok && !it.Ok {
		return "ΓÜá Ch╞░a ph├ít hiß╗çn GPU hß╗ù trß╗ú ΓÇö Kiß╗âm tra khe cß║»m & driver tr╞░ß╗¢c khi c├ái ─æß║╖t"
	}
	var warns []string
	if it, ok := m["Secure Boot"]; ok && !it.Ok {
		warns = append(warns, "Secure Boot ─æang bß║¡t (cß║ºn v├áo BIOS tß║»t)")
	}
	if it, ok := m["Chß║┐ ─æß╗Ö Boot"]; ok && !it.Ok {
		warns = append(warns, "Khß╗ƒi ─æß╗Öng Legacy+MBR (cß║ºn d├╣ng mbr2gpt chuyß╗ân sang GPT)")
	}
	if it, ok := m["Driver PCIe (Ch╞░a c├ái)"]; ok && !it.Ok {
		warns = append(warns, "Driver PCIe ch╞░a ─æ╞░ß╗úc c├ái ─æß║╖t")
	}
	if it, ok := m["T├íc vß╗Ñ tß╗▒ khß╗ƒi ─æß╗Öng"]; ok && !it.Ok {
		warns = append(warns, "T├íc vß╗Ñ tß╗▒ khß╗ƒi ─æß╗Öng PCIe ch╞░a ─æ─âng k├╜")
	}
	// σÉ»σè¿Θí╣: Σ╗à UEFI Σ╕ïµúÇµƒÑ
	if it, ok := m["Mß╗Ñc khß╗ƒi ─æß╗Öng BIOS"]; ok && !it.Ok {
		if bl, ok2 := m["Chß║┐ ─æß╗Ö Boot"]; !ok2 || bl.Ok {
			if strings.Contains(it.Note, "ch╞░a ─æß║╖t") {
				warns = append(warns, "Mß╗Ñc khß╗ƒi ─æß╗Öng tß╗ôn tß║íi nh╞░ng ch╞░a ─æß║╖t ─æß║ºu ti├¬n")
			} else {
				warns = append(warns, "Ch╞░a tß║ío mß╗Ñc khß╗ƒi ─æß╗Öng UEFI")
			}
		}
	}
	if len(warns) == 0 {
		return "Γ£ô M├┤i tr╞░ß╗¥ng sß║╡n s├áng ΓÇö ─É├ú tß╗▒ chß╗ìn mß╗Ñc cß║ºn thiß║┐t, bß║Ñm [C├ái ─æß║╖t mß╗Ñc ─æ├ú chß╗ìn] hoß║╖c [C├ái ─æß║╖t to├án bß╗Ö]"
	}
	s := "ΓÜá " + strings.Join(warns, "; ")
	if r := []rune(s); len(r) > 90 {
		s = string(r[:90]) + "ΓÇª"
	}
	return s
}

// envGuide: "Known issues -> Resolution steps"
func (st *guiState) envGuide(items []statusItem) string {
	m := map[string]statusItem{}
	for _, it := range items {
		m[it.Name] = it
	}
	var g []string
	if it, ok := m["Card ─æß╗ô hoß║í"]; ok && !it.Ok {
		g = append(g, "┬╖ Ch╞░a ph├ít hiß╗çn GPU: Γæá Kiß╗âm tra nguß╗ôn phß╗Ñ & cß║»m chß║»c khe PCIe; Γæí Xem Device Manager c├│ Code 43 kh├┤ng; Γæó Tß║»t CSM trong BIOS (chß╗ìn UEFI thuß║ºn)")
	}
	if it, ok := m["Secure Boot"]; ok && !it.Ok {
		g = append(g, "┬╖ Secure Boot ─æang bß║¡t: Khß╗ƒi ─æß╗Öng lß║íi bß║Ñm Del/F2 v├áo BIOS ΓåÆ Security/Boot ΓåÆ Secure Boot=Disabled ΓåÆ F10 l╞░u v├á khß╗ƒi ─æß╗Öng lß║íi")
	}
	if it, ok := m["Chß║┐ ─æß╗Ö Boot"]; ok && !it.Ok {
		g = append(g, "┬╖ Khß╗ƒi ─æß╗Öng Legacy+MBR: Kh├┤ng c├│ ph├ón v├╣ng EFI ΓåÆ Mß╗ƒ CMD Admin chß║íy: mbr2gpt /validate /allowfullos ΓåÆ mbr2gpt /convert /allowfullos ΓåÆ V├áo BIOS tß║»t CSM")
	}
	if it, ok := m["Mß╗Ñc khß╗ƒi ─æß╗Öng BIOS"]; ok && !it.Ok {
		if bl, ok2 := m["Chß║┐ ─æß╗Ö Boot"]; !ok2 || bl.Ok {
			if strings.Contains(it.Note, "ch╞░a ─æß║╖t") {
				g = append(g, "┬╖ Mß╗Ñc khß╗ƒi ─æß╗Öng ch╞░a ╞░u ti├¬n: V├áo BIOS ─æß║╖t '40HX Unlock' l├¬n vß╗ï tr├¡ ─æß║ºu ti├¬n (Boot Option #1)")
			} else {
				g = append(g, "┬╖ Ch╞░a tß║ío mß╗Ñc khß╗ƒi ─æß╗Öng: T├¡ch chß╗ìn [EFI Mß╗ƒ kho├í + Mß╗Ñc khß╗ƒi ─æß╗Öng BIOS] ─æß╗â tß║ío tß╗▒ ─æß╗Öng")
			}
		}
	}
	if it, ok := m["Driver PCIe (Ch╞░a c├ái)"]; ok && !it.Ok {
		g = append(g, "┬╖ Driver PCIe ch╞░a c├ái: T├¡ch chß╗ìn [C├ái ─æß║╖t Driver PCIe + Th├¬m loß║íi trß╗½ Defender] ─æß╗â c├ái ─æß║╖t")
	}
	if it, ok := m["T├íc vß╗Ñ tß╗▒ khß╗ƒi ─æß╗Öng"]; ok && !it.Ok {
		g = append(g, "┬╖ T├íc vß╗Ñ tß╗▒ mß╗ƒ kho├í ch╞░a ─æ─âng k├╜: T├¡ch chß╗ìn [Tß╗▒ khß╗ƒi ─æß╗Öng mß╗ƒ kho├í PCIe khi ─æ─âng nhß║¡p]")
	}
	if it, ok := m["ESP EFI Mß╗ƒ kho├í"]; ok && !it.Ok {
		if bl, ok2 := m["Chß║┐ ─æß╗Ö Boot"]; !ok2 || bl.Ok {
			g = append(g, "┬╖ EFI mß╗ƒ kho├í ch╞░a nß║íp: T├¡ch chß╗ìn [EFI Mß╗ƒ kho├í + Mß╗Ñc khß╗ƒi ─æß╗Öng BIOS] (chß╗ë d├ánh cho CMP 40HX)")
		}
	}
	return strings.Join(g, "\n")
}

// scanOnce: σÅ¬Φ»╗µë½µÅÅΣ╕Çµ¼íσ╣╢σê╖µû░Θí╢Θâ¿µÅÉτñ║(Σ╕ìσ▒òτñ║µÿÄτ╗å)πÇé
func (st *guiState) scanOnce() {
	items := scanStatus()
	st.lastScan = items
	tip := st.summaryText(items)
	st.sync(func() {
		if st.tip != nil {
			st.tip.SetText(tip)
		}
	})
	av := hxcore.DetectThirdPartyAV()
	avKey := strings.Join(av, ",")
	if avKey != st.lastAV {
		if len(av) > 0 {
			fmt.Println("[Antivirus] Ph├ít hiß╗çn phß║ºn mß╗üm bß║úo mß║¡t b├¬n thß╗⌐ ba: " + strings.Join(av, " / ") +
				" ΓÇö Vui l├▓ng th├¬m 4 ─æ╞░ß╗¥ng dß║½n driver v├áo danh s├ích tin cß║¡y/loß║íi trß╗½:")
			fmt.Println("        C:\\Windows\\System32\\drivers\\ThrottleStop.sys")
			fmt.Println("        C:\\Windows\\System32\\drivers\\WinRing0x64.sys")
			fmt.Println("        %ProgramData%\\40HXUnlock\\drivers\\ThrottleStop.sys v├á WinRing0x64.sys")
		}
		st.lastAV = avKey
	}
}

// applySmartDefaults: µîëµ£ÇΦ┐æΣ╕Çµ¼íµë½µÅÅΘóäσï╛ΘÇë ΓÇö τ╗äΣ╗╢τ╝║σñ▒/µ£¬Φ╛╛µáçµëìσï╛(σ╖▓ΦúàΣ╕ìσï╛=Σ╕ìΦªåτ¢û)πÇé
func (st *guiState) applySmartDefaults() {
	items := st.lastScan
	if len(items) == 0 {
		items = scanStatus()
		st.lastScan = items
	}
	flags := map[string]bool{}
	for _, it := range items {
		flags[it.Name] = it.Ok
	}
	if !flags["Card ─æß╗ô hoß║í"] {
		fmt.Println("[i] Ch╞░a ph├ít hiß╗çn card ─æß╗ô hoß║í hß╗ù trß╗ú ΓÇö Giß╗» nguy├¬n kh├┤ng chß╗ìn mß╗Ñc n├áo (vui l├▓ng kiß╗âm tra GPU/driver)")
		st.sync(func() {
			st.ckGsp.SetChecked(false)
			st.ckDrv.SetChecked(false)
			st.ckEfi.SetChecked(false)
			st.ckTask.SetChecked(false)
			st.ckFast.SetChecked(false)
			st.ckAspm.SetChecked(false)
			st.ckPerf.SetChecked(false)
			st.ckDefOff.SetChecked(false)
		})
		return
	}
	aspmOK := true
	if v, present := flags["Tiß║┐t kiß╗çm ─æiß╗çn PCIe (ASPM)"]; present {
		aspmOK = v
	}
	needGsp := !flags["GSP (EnableGpuFirmware)"]
	needDrv := hxcore.Gen2DriversNeedDeploy()
	needEfi := false
	if flags["Chß║┐ ─æß╗Ö Boot"] {
		needEfi = !flags["ESP EFI Mß╗ƒ kho├í"] || !flags["Mß╗Ñc khß╗ƒi ─æß╗Öng BIOS"]
	} else {
		fmt.Println("[i] Khß╗ƒi ─æß╗Öng Legacy+MBR: Kh├┤ng thß╗â c├ái EFI mß╗ƒ kho├í ΓÇö Bß╗Å chß╗ìn (cß║ºn mbr2gpt tr╞░ß╗¢c)")
	}
	needTask := !flags["T├íc vß╗Ñ tß╗▒ khß╗ƒi ─æß╗Öng"]
	needFast := !flags["Khß╗ƒi ─æß╗Öng nhanh (Fast Startup)"]
	needAspm := !aspmOK
	needPerf := !hxcore.HighPerfPlanActive()
	needDefOff, defKnown := false, false
	if on, err := hxcore.DefenderRealtimeProtectionOn(); err == nil {
		defKnown = true
		needDefOff = on
	} else {
		defWarn := "M├íy t├¡nh kh├┤ng c├│ module quß║ún trß╗ï Defender (phß║ºn mß╗üm diß╗çt virus b├¬n thß╗⌐ 3 vui l├▓ng tß╗▒ th├¬m whitelist)"
		if !errors.Is(err, hxcore.ErrMpUnavailable) {
			defWarn = "Kiß╗âm tra trß║íng th├íi Defender thß║Ñt bß║íi: " + err.Error()
		}
		if defWarn != st.lastDefWarn {
			fmt.Println("[i] " + defWarn)
			st.lastDefWarn = defWarn
		}
	}
	var pre []string
	st.sync(func() {
		st.ckGsp.SetChecked(needGsp)
		if needGsp {
			pre = append(pre, "GSP")
		}
		st.ckDrv.SetChecked(needDrv)
		if needDrv {
			pre = append(pre, "Driver PCIe")
		}
		st.ckEfi.SetChecked(needEfi)
		if needEfi {
			pre = append(pre, "EFI Mß╗ƒ kho├í + Khß╗ƒi ─æß╗Öng BIOS")
		}
		st.ckTask.SetChecked(needTask)
		if needTask {
			pre = append(pre, "Tß╗▒ mß╗ƒ kho├í khi ─æ─âng nhß║¡p")
		}
		st.ckFast.SetChecked(needFast)
		if needFast {
			pre = append(pre, "Tß║»t Fast Startup")
		}
		st.ckAspm.SetChecked(needAspm)
		if needAspm {
			pre = append(pre, "Tß║»t ASPM")
		}
		st.ckPerf.SetChecked(needPerf)
		if needPerf {
			pre = append(pre, "Hiß╗çu n─âng cao High Performance")
		}
		wantDefOff := defKnown && needDefOff
		st.ckDefOff.SetChecked(wantDefOff)
		if wantDefOff {
			pre = append(pre, "Tß║»t Defender Realtime")
		}
	})
	if len(pre) > 0 {
		fmt.Println("[i] Tß╗▒ ─æß╗Öng chß╗ìn: " + strings.Join(pre, " / ") + " ΓåÆ Bß║Ñm [C├ái ─æß║╖t mß╗Ñc ─æ├ú chß╗ìn] ─æß╗â thß╗▒c thi; C├íc mß╗Ñc ─æ├ú sß║╡n s├áng ─æ╞░ß╗úc bß╗Å qua")
		return
	}
	var ready []string
	if !needGsp {
		ready = append(ready, "GSP ─æ├ú bß║¡t")
	}
	if !needDrv {
		ready = append(ready, "Driver PCIe ─æ├ú sß║╡n s├áng")
	}
	if !flags["Chß║┐ ─æß╗Ö Boot"] {
		ready = append(ready, "EFI Mß╗ƒ kho├í (cß║ºn mbr2gpt tr╞░ß╗¢c)")
	} else if !needEfi {
		ready = append(ready, "EFI + Khß╗ƒi ─æß╗Öng BIOS ─æ├ú sß║╡n s├áng")
	}
	if !needTask {
		ready = append(ready, "Tß╗▒ khß╗ƒi ─æß╗Öng ─æ├ú ─æ─âng k├╜")
	}
	if !needFast {
		ready = append(ready, "Khß╗ƒi ─æß╗Öng nhanh ─æ├ú tß║»t")
	}
	if !needAspm {
		ready = append(ready, "ASPM ─æ├ú tß║»t")
	}
	if !needPerf {
		ready = append(ready, "─É├ú ß╗ƒ chß║┐ ─æß╗Ö High Performance")
	}
	if defKnown && !needDefOff {
		ready = append(ready, "Defender Realtime ─æ├ú tß║»t")
	}
	fmt.Println("[i] Tß║Ñt cß║ú ─æ├ú sß║╡n s├áng, kh├┤ng cß║ºn chß╗ìn th├¬m: " + strings.Join(ready, " | "))
}

// printDefErr: Hiß╗ân thß╗ï lß╗ùi Defender ngß║»n gß╗ìn.
func printDefErr(prefix string, err error) {
	if err == nil {
		return
	}
	if errors.Is(err, hxcore.ErrMpUnavailable) {
		fmt.Println(prefix + "Hß╗ç thß╗æng kh├┤ng c├│ module quß║ún trß╗ï Defender ΓÇö Th├¬m loß║íi trß╗½ / tß║»t bß║úo vß╗ç tß╗▒ ─æß╗Öng kh├┤ng khß║ú dß╗Ñng")
		fmt.Println(prefix + "Nß║┐u d├╣ng phß║ºn mß╗üm diß╗çt virus b├¬n thß╗⌐ ba, vui l├▓ng th├¬m ngoß║íi lß╗ç cho:")
		fmt.Println("      C:\\Windows\\System32\\drivers\\ThrottleStop.sys / WinRing0x64.sys")
		fmt.Println("      %ProgramData%\\40HXUnlock\\drivers\\ (2 file .sys c├╣ng t├¬n)")
		return
	}
	fmt.Println(prefix + err.Error())
}

// loadPolicyUI: Khß╗ƒi ─æß╗Öng ─æß╗ìc lß║íi ch├¡nh s├ích (gß╗ìi tr├¬n luß╗ông UI ΓÇö sau Create, tr╞░ß╗¢c Run)
func (st *guiState) loadPolicyUI() {
	strat := hxcore.DriverStrategy()
	for i, rb := range st.rbStrategy {
		rb.SetChecked(i == strat)
	}
	st.ckAutoHard.SetChecked(hxcore.ConfigInt("Gen2AutoHard", 1) != 0)
	cnt, interval := hxcore.Gen2RetryPolicy()
	st.neRetryCnt.SetValue(float64(cnt))
	st.neRetryMin.SetValue(float64(interval))
}

// savePolicy: L╞░u cß║Ñu h├¼nh PCIe (HKLM\SOFTWARE\40HXUnlock, ─æß╗ìc bß╗ƒi t├íc vß╗Ñ tß╗▒ chß║íy v├á lß╗çnh -gen2)
func (st *guiState) savePolicy() {
	defer st.end()
	strat := 0
	for i, rb := range st.rbStrategy {
		if rb.Checked() {
			strat = i
		}
	}
	if err := hxcore.SetConfigInt("DriverStrategy", strat); err != nil {
		fmt.Println("[Cß║Ñu h├¼nh] L╞░u thß║Ñt bß║íi:", err)
		return
	}
	auto := 0
	if st.ckAutoHard.Checked() {
		auto = 1
	}
	cnt, interval := int(st.neRetryCnt.Value()), int(st.neRetryMin.Value())
	hxcore.SetConfigInt("Gen2AutoHard", auto)
	hxcore.SetConfigInt("Gen2RetryCount", cnt)
	hxcore.SetConfigInt("Gen2RetryIntervalMin", interval)
	fmt.Printf("[Cß║Ñu h├¼nh] ─É├ú l╞░u: Chiß║┐n l╞░ß╗úc driver=%d Gen2AutoHard=%d Thß╗¡ lß║íi=%d lß║ºn / Gi├ún c├ích=%d ph├║t\n", strat, auto, cnt, interval)
	if strat == hxcore.DriverStrategyResident {
		if ok, _, _ := hxcore.TaskInfo(gen2TaskName); !ok {
			fmt.Println("[Gß╗úi ├╜] Chß║┐ ─æß╗Ö Th╞░ß╗¥ng tr├║ cß║ºn t├íc vß╗Ñ tß╗▒ chß║íy khi ─æ─âng nhß║¡p: Vui l├▓ng t├¡ch chß╗ìn [Tß╗▒ ─æß╗Öng mß╗ƒ kho├í PCIe khi ─æ─âng nhß║¡p Windows] ß╗ƒ mß╗Ñc Γæá rß╗ôi bß║Ñm [C├ái ─æß║╖t mß╗Ñc ─æ├ú chß╗ìn], hoß║╖c bß║Ñm n├║t [Mß╗ƒ kho├í ngay & C├ái tß╗▒ khß╗ƒi ─æß╗Öng]")
		} else {
			fmt.Println("[Gß╗úi ├╜] ─É├ú chß╗ìn Th╞░ß╗¥ng tr├║: Vui l├▓ng bß║Ñm [Mß╗ƒ kho├í ngay & C├ái tß╗▒ khß╗ƒi ─æß╗Öng] ─æß╗â cß║¡p nhß║¡t tham sß╗æ canh giß╗» (-guard) cho t├íc vß╗Ñ")
		}
	}
}

func runGUI() {
	if !isAdmin() {
		selfElevate()
		return
	}
	st := &guiState{log: &guiLog{}}

	createErr := MainWindow{
		AssignTo: &st.mw,
		Title:    "Tr├¼nh Mß╗ƒ Kho├í CMP 40HX / 30HX v3.0.0",
		MinSize:  Size{Width: 800, Height: 680},
		Size:     Size{Width: 880, Height: 800},
		Layout:   VBox{Spacing: 6},
		Children: []Widget{
			GroupBox{
				Title:  "Γæá C├ái ─æß║╖t th├ánh phß║ºn & M├┤i tr╞░ß╗¥ng (Tß╗▒ ─æß╗Öng chß╗ìn theo m├íy; T├¡ch chß╗ìn = C├ái ─æß║╖t/L├ám mß╗¢i)",
				Layout: VBox{Spacing: 4},
				Children: []Widget{
					Label{AssignTo: &st.tip, Text: "─Éang kiß╗âm tra m├┤i tr╞░ß╗¥ng hß╗ç thß╗ængΓÇª"},
					Composite{
						Layout: Grid{Columns: 2},
						Children: []Widget{
							CheckBox{AssignTo: &st.ckGsp, Text: "Bß║¡t GSP (EnableGpuFirmware=1)"},
							CheckBox{AssignTo: &st.ckEfi, Text: "EFI Mß╗ƒ kho├í + Mß╗Ñc khß╗ƒi ─æß╗Öng BIOS (Chß╗ë 40HX)"},
							CheckBox{AssignTo: &st.ckDrv, Text: "C├ái ─æß║╖t Driver PCIe + Th├¬m loß║íi trß╗½ Defender"},
							CheckBox{AssignTo: &st.ckTask, Text: "Tß╗▒ ─æß╗Öng mß╗ƒ kho├í PCIe khi ─æ─âng nhß║¡p Windows"},
							CheckBox{AssignTo: &st.ckFast, Text: "Nguß╗ôn: Tß║»t Fast Startup (Khß╗ƒi ─æß╗Öng nhanh)"},
							CheckBox{AssignTo: &st.ckAspm, Text: "Nguß╗ôn: Tß║»t ASPM (Tiß║┐t kiß╗çm ─æiß╗çn PCIe)"},
							CheckBox{AssignTo: &st.ckPerf, Text: "Nguß╗ôn: Bß║¡t chß║┐ ─æß╗Ö High Performance"},
							CheckBox{AssignTo: &st.ckDefOff, Text: "Tß║»t bß║úo vß╗ç thß╗¥i gian thß╗▒c Defender"},
						},
					},
					Composite{
						Layout: HBox{},
						Children: []Widget{
							PushButton{AssignTo: &st.pbInstall, Text: "C├ái ─æß║╖t mß╗Ñc ─æ├ú chß╗ìn", OnClicked: func() {
								if !st.begin("C├ái ─æß║╖t th├ánh phß║ºn") {
									return
								}
								sel := map[string]bool{
									"gsp":    st.ckGsp.Checked(),
									"drv":    st.ckDrv.Checked(),
									"efi":    st.ckEfi.Checked(),
									"task":   st.ckTask.Checked(),
									"fast":   st.ckFast.Checked(),
									"aspm":   st.ckAspm.Checked(),
									"perf":   st.ckPerf.Checked(),
									"defoff": st.ckDefOff.Checked(),
								}
								go st.installSelected(sel)
							}},
							PushButton{AssignTo: &st.pbFull, Text: "C├ái ─æß║╖t to├án bß╗Ö (Mß╗Öt chß║ím)", OnClicked: func() {
								if !st.begin("C├ái ─æß║╖t to├án bß╗Ö") {
									return
								}
								go func() {
									defer st.end()
									st.sync(func() { st.pbFull.SetEnabled(false) })
									defer st.sync(func() { st.pbFull.SetEnabled(true) })
									install()
									st.scanOnce()
									st.applySmartDefaults()
								}()
							}},
						},
					},
				},
			},
			GroupBox{
				Title:  "Γæí Cß║Ñu h├¼nh PCIe (L╞░u c├│ hiß╗çu lß╗▒c ngay; T├íc vß╗Ñ tß╗▒ chß║íy sß║╜ ├íp dß╗Ñng cß║Ñu h├¼nh n├áy)",
				Layout: VBox{Spacing: 4},
				Children: []Widget{
					Label{Text: "Ch├¡nh s├ích Driver: C├ích xß╗¡ l├╜ driver can thiß╗çp sau khi mß╗ƒ kho├í PCIe"},
					Label{Text: "Γæá D├╣ng xong gß╗í ngay (Mß║╖c ─æß╗ïnh, sß║ích sß║╜, chuß║⌐n game/anti-cheat)   Γæí Thß╗¡ lß║íi nß║┐u lß╗ùi   Γæó Th╞░ß╗¥ng tr├║ (Giß╗» driver, canh v├á giß╗» tß╗æc ─æß╗Ö)"},
					Composite{
						Layout: Grid{Columns: 3},
						Children: []Widget{
							RadioButton{AssignTo: &st.rbStrategy[0], Text: "D├╣ng xong gß╗í ngay (Khuy├¬n d├╣ng)"},
							RadioButton{AssignTo: &st.rbStrategy[1], Text: "Tß╗▒ ─æß╗Öng thß╗¡ lß║íi khi lß╗ùi"},
							RadioButton{AssignTo: &st.rbStrategy[2], Text: "Th╞░ß╗¥ng tr├║ (Canh giß╗» PCIe)"},
						},
					},
					CheckBox{AssignTo: &st.ckAutoHard, Text: "Tß╗▒ ─æß╗Öng k├¡ch hoß║ít Stage 2 nß║┐u ch╞░a ─æß║ít (Link Disable + Kh├┤i phß╗Ñc PnP)"},
					Composite{
						Layout: HBox{},
						Children: []Widget{
							Label{Text: "Thß╗¡ lß║íi khi lß╗ùi:"},
							NumberEdit{AssignTo: &st.neRetryCnt, MinValue: 0.0, MaxValue: 12.0, MinSize: Size{Width: 56}},
							Label{Text: "lß║ºn / Gi├ún c├ích:"},
							NumberEdit{AssignTo: &st.neRetryMin, MinValue: 1.0, MaxValue: 240.0, MinSize: Size{Width: 56}},
							Label{Text: "ph├║t"},
						},
					},
					Composite{
						Layout: HBox{},
						Children: []Widget{
							PushButton{AssignTo: &st.pbSave, Text: "L╞░u cß║Ñu h├¼nh", OnClicked: func() {
								if !st.begin("L╞░u cß║Ñu h├¼nh") {
									return
								}
								go st.savePolicy()
							}},
							PushButton{AssignTo: &st.pbGen2, Text: "Mß╗ƒ kho├í PCIe ngay (Phi├¬n n├áy)", OnClicked: func() {
								if !st.begin("Mß╗ƒ kho├í PCIe ngay") {
									return
								}
								go func() {
									defer st.end()
									st.sync(func() { st.pbGen2.SetEnabled(false) })
									defer st.sync(func() { st.pbGen2.SetEnabled(true) })
									fmt.Println("[PCIe] Mß╗ƒ kho├í ngay mß╗Öt lß║ºn (t╞░╞íng ─æ╞░╞íng lß╗çnh -gen2)...")
									gen2Main()
									st.scanOnce()
								}()
							}},
							PushButton{AssignTo: &st.pbGen2Install, Text: "Mß╗ƒ kho├í ngay & C├ái tß╗▒ khß╗ƒi ─æß╗Öng", OnClicked: func() {
								if !st.begin("Mß╗ƒ kho├í & C├ái tß╗▒ khß╗ƒi ─æß╗Öng") {
									return
								}
								go func() {
									defer st.end()
									st.sync(func() { st.pbGen2Install.SetEnabled(false) })
									defer st.sync(func() { st.pbGen2Install.SetEnabled(true) })
									fmt.Println("== Mß╗ƒ kho├í PCIe v├á c├ái ─æß║╖t tß╗▒ khß╗ƒi ─æß╗Öng ==")
									fmt.Println("[PCIe] B╞░ß╗¢c 1/2: Mß╗ƒ kho├í phi├¬n hiß╗çn tß║íi...")
									gen2Main()
									fmt.Println("[PCIe] B╞░ß╗¢c 2/2: C├ái ─æß║╖t tß╗▒ khß╗ƒi ─æß╗Öng khi ─æ─âng nhß║¡p Windows...")
									installDrivers()
									if err := hxcore.AddDefenderExclusions(); err != nil {
										printDefErr("  [Defender] ", err)
									} else {
										fmt.Println("  [Defender] ─É├ú th├¬m loß║íi trß╗½ cho file driver v├á ProgramData")
									}
									setRunKey()
									if err := setupGen2Task(); err != nil {
										fmt.Println("  [!] ─É─âng k├╜ t├íc vß╗Ñ tß╗▒ chß║íy thß║Ñt bß║íi:", err)
									} else {
										fmt.Println("  [Tß╗▒ chß║íy] ─É─âng k├╜ th├ánh c├┤ng ΓÇö M├íy t├¡nh sß║╜ tß╗▒ ─æß╗Öng mß╗ƒ kho├í PCIe khi ─æ─âng nhß║¡p Windows")
									}
									st.scanOnce()
								}()
							}},
						},
					},
				},
			},
			GroupBox{
				Title:  "Γæó Nhß║¡t k├╜ hoß║ít ─æß╗Öng (Thß╗¥i gian thß╗▒c)",
				Layout: VBox{},
				Children: []Widget{
					TextEdit{AssignTo: &st.teLog, ReadOnly: true, VScroll: true,
						MinSize: Size{Height: 120}, StretchFactor: 2},
				},
			},
		},
	}.Create()
	if createErr != nil {
		msgbox("Tr├¼nh C├ái ─Éß║╖t 40HX / 30HX", "Khß╗ƒi tß║ío GUI thß║Ñt bß║íi: "+createErr.Error()+"\nVui l├▓ng sß╗¡ dß╗Ñng chß║┐ ─æß╗Ö d├▓ng lß╗çnh (40HXInstaller.exe -h).", mbIconError)
		return
	}
	st.log.mw = st.mw
	st.log.te = st.teLog
	AttachLogSink(st.log)

	st.loadPolicyUI()

	fmt.Println("Tr├¼nh Mß╗ƒ Kho├í CMP 40HX / 30HX v3.0.0 ─æ├ú khß╗ƒi ─æß╗Öng (Quyß╗ün Admin).")
	go func() {
		// µëôσ╝ÇΦç¬σè¿µë½µÅÅΣ╕Çµ¼í: Θóäσï╛ΘÇë + Θí╢Θâ¿µÅÉτñ║; µ£ƒΘù┤τªüτö¿µëºΦíîµîëΘÆ«Θÿ▓σ╣╢σÅæπÇé
		// τ╗ôµ₧£τö▒ applySmartDefaults µëôσì░([i] σ╖▓Θóäσï╛ΓÇª / [i] τ╗äΣ╗╢σ¥çσ╖▓σ░▒τ╗¬ΓÇª)πÇé
		st.setActionsEnabled(false)
		defer st.setActionsEnabled(true)
		st.scanOnce()
		st.applySmartDefaults()
	}()
	st.mw.Run()
}

// executeInstallSelected: Thß╗▒c hiß╗çn c├ái ─æß║╖t c├íc th├ánh phß║ºn ─æ├ú chß╗ìn
func executeInstallSelected(sel map[string]bool) {
	nameOf := map[string]string{
		"gsp": "Bß║¡t GSP", "drv": "C├ái ─æß║╖t Driver PCIe", "efi": "EFI Mß╗ƒ kho├í + Khß╗ƒi ─æß╗Öng",
		"task": "Tß╗▒ mß╗ƒ kho├í khi ─æ─âng nhß║¡p", "fast": "Tß║»t Khß╗ƒi ─æß╗Öng nhanh", "aspm": "Tß║»t ASPM",
		"perf": "Hiß╗çu n─âng cao (High Perf)", "defoff": "Tß║»t Defender thß╗¥i gian thß╗▒c",
	}
	var parts []string
	for _, k := range []string{"gsp", "drv", "efi", "task", "fast", "aspm", "perf", "defoff"} {
		if sel[k] {
			parts = append(parts, nameOf[k])
		}
	}
	if len(parts) == 0 {
		fmt.Println("[i] Ch╞░a chß╗ìn th├ánh phß║ºn n├áo ΓÇö Vui l├▓ng t├¡ch chß╗ìn tr╞░ß╗¢c khi bß║Ñm [C├ái ─æß║╖t mß╗Ñc ─æ├ú chß╗ìn]")
		return
	}
	fmt.Println("== Thß╗▒c hiß╗çn: " + strings.Join(parts, " / ") + " ==")
	if sel["gsp"] {
		fmt.Println("ΓöÇΓöÇΓöÇΓöÇ Bß║¡t GSP (EnableGpuFirmware=1, then chß╗æt chß╗æng ─æen m├án h├¼nh) ΓöÇΓöÇΓöÇΓöÇ")
		if err := enableGsp(); err != nil {
			fmt.Println("  [GSP] Thß║Ñt bß║íi:", err)
		} else {
			fmt.Println("  [GSP] EnableGpuFirmware=1 ─É├ú thiß║┐t lß║¡p (Khß╗ƒi ─æß╗Öng lß║íi m├íy ─æß╗â GSP-RM c├│ hiß╗çu lß╗▒c)")
		}
	}
	if sel["drv"] {
		fmt.Println("ΓöÇΓöÇΓöÇΓöÇ C├ái ─æß║╖t Driver PCIe + Th├¬m loß║íi trß╗½ Defender ΓöÇΓöÇΓöÇΓöÇ")
		installDrivers()
		if err := hxcore.AddDefenderExclusions(); err != nil {
			printDefErr("  [Defender] ", err)
		} else {
			fmt.Println("  [Defender] ─É├ú th├¬m th╞░ mß╗Ñc driver v├á ProgramData v├áo danh s├ích loß║íi trß╗½")
		}
	}
	if sel["efi"] {
		fmt.Println("ΓöÇΓöÇΓöÇΓöÇ Triß╗ân khai EFI Mß╗ƒ kho├í + Mß╗Ñc khß╗ƒi ─æß╗Öng BIOS (Chß╗ë 40HX) ΓöÇΓöÇΓöÇΓöÇ")
		installEFI()
	}
	if sel["task"] {
		fmt.Println("ΓöÇΓöÇΓöÇΓöÇ Tß╗▒ ─æß╗Öng mß╗ƒ kho├í PCIe khi ─æ─âng nhß║¡p (T├íc vß╗Ñ SYSTEM + Run key) ΓöÇΓöÇΓöÇΓöÇ")
		setRunKey()
		if err := setupGen2Task(); err != nil {
			fmt.Println("  [!] ─É─âng k├╜ t├íc vß╗Ñ tß╗▒ chß║íy thß║Ñt bß║íi:", err)
			fmt.Println("  [!] C├│ thß╗â chß║íy bß║▒ng tay vß╗¢i quyß╗ün Admin: 40HXInstaller.exe -task")
		}
	}
	if sel["fast"] || sel["aspm"] || sel["perf"] {
		fmt.Println("ΓöÇΓöÇΓöÇΓöÇ C├ái ─æß║╖t Nguß╗ôn ─æiß╗çn (C├│ thß╗â bß║¡t lß║íi trong Windows Power Options) ΓöÇΓöÇΓöÇΓöÇ")
	}
	if sel["fast"] {
		if hxcore.FastStartupOn() {
			if err := hxcore.SetFastStartupOff(); err != nil {
				fmt.Println("  [Nguß╗ôn] Tß║»t Khß╗ƒi ─æß╗Öng nhanh (Fast Startup) thß║Ñt bß║íi:", err)
			} else {
				fmt.Println("  [Nguß╗ôn] ─É├ú tß║»t Khß╗ƒi ─æß╗Öng nhanh (HiberbootEnabled=0)")
			}
		} else {
			fmt.Println("  [Nguß╗ôn] Khß╗ƒi ─æß╗Öng nhanh: ─É├ú tß║»t tß╗½ tr╞░ß╗¢c (OK)")
		}
	}
	if sel["aspm"] {
		if ac, dc, ok := hxcore.ASPMSavings(); !ok {
			fmt.Println("  [Nguß╗ôn] PCIe ASPM: M├íy t├¡nh kh├┤ng hß╗ù trß╗ú c├ái ─æß║╖t n├áy, bß╗Å qua")
		} else if ac == 0 && dc == 0 {
			fmt.Println("  [Nguß╗ôn] PCIe ASPM: ─É├ú tß║»t tß╗½ tr╞░ß╗¢c (OK)")
		} else {
			if err := hxcore.SetASPMOff(); err != nil {
				fmt.Println("  [Nguß╗ôn] Tß║»t ASPM thß║Ñt bß║íi:", err)
			} else {
				fmt.Printf("  [Nguß╗ôn] PCIe ASPM ─æ├ú tß║»t (Gß╗æc: AC=%d/DC=%d)\n", ac, dc)
			}
		}
	}
	if sel["perf"] {
		if hxcore.HighPerfPlanActive() {
			fmt.Println("  [Nguß╗ôn] Chß║┐ ─æß╗Ö nguß╗ôn: ─É├ú l├á High Performance (OK)")
		} else if err := hxcore.SetHighPerfPlan(); err != nil {
			fmt.Println("  [Nguß╗ôn] Chuyß╗ân sang High Performance thß║Ñt bß║íi:", err)
		} else {
			fmt.Println("  [Nguß╗ôn] ─É├ú chuyß╗ân sang chß║┐ ─æß╗Ö High Performance")
		}
	}
	if sel["defoff"] {
		fmt.Println("ΓöÇΓöÇΓöÇΓöÇ Bß║úo vß╗ç thß╗¥i gian thß╗▒c Windows Defender ΓöÇΓöÇΓöÇΓöÇ")
		on, err := hxcore.DefenderRealtimeProtectionOn()
		if err != nil {
			printDefErr("  [!] ", err)
		} else if !on {
			fmt.Println("  [Defender] Bß║úo vß╗ç thß╗¥i gian thß╗▒c hiß╗çn ─æang Tß║»t (Kh├┤ng cß║ºn thao t├íc)")
		} else if err := hxcore.SetDefenderRealtimeProtection(false); err != nil {
			printDefErr("  [!] ", err)
			if !errors.Is(err, hxcore.ErrMpUnavailable) {
				fmt.Println("  [!] Nguy├¬n nh├ón th╞░ß╗¥ng gß║╖p: T├¡nh n─âng 'Tamper Protection' ─æang bß║¡t ΓÇö Vui l├▓ng tß║»t trong Windows Security rß╗ôi thß╗¡ lß║íi")
			}
		} else {
			fmt.Println("  [Defender] ─É├ú tß║»t bß║úo vß╗ç thß╗¥i gian thß╗▒c")
			fmt.Println("  [Defender] Bß║¡t lß║íi: Chß║íy PowerShell Admin: Set-MpPreference -DisableRealtimeMonitoring $False")
		}
	}
}

// installSelected: C├ái ─æß║╖t c├íc th├ánh phß║ºn v├á thiß║┐t lß║¡p ─æ├ú chß╗ìn (thß╗▒c hiß╗çn tuß║ºn tß╗▒, ─æ╞░a v├áo bß║úng log).
func (st *guiState) installSelected(sel map[string]bool) {
	defer st.end()
	st.sync(func() { st.pbInstall.SetEnabled(false) })
	defer st.sync(func() { st.pbInstall.SetEnabled(true) })
	executeInstallSelected(sel)
	fmt.Println("== Ho├án tß║Ñt thß╗▒c hiß╗çn, tß╗▒ ─æß╗Öng qu├⌐t lß║íi trß║íng th├íi ==")
	st.scanOnce()
	st.applySmartDefaults()
}