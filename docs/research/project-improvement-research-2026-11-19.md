# Nghiên cứu cải thiện dự án CMP 30HX/40HX Unlock

Ngày nghiên cứu: **2026-11-19**  
Phạm vi: working tree hiện tại; chỉ đọc mã, tài liệu và nguồn sơ cấp. Không chạy thao tác Registry, driver, EFI, reboot, PCIe/MMIO hay kiểm thử trên phần cứng.

## Kết luận ngắn

Ba việc nên làm trước khi phát hành tiếp theo:

1. **P0 — khóa chặt Web UI HTTP cũ chạy quyền Administrator**: server loopback trong `inst40hx` có API thay đổi hệ thống nhưng chưa kiểm tra method/origin/host; cùng repo đã có guard và test phù hợp trong fallback server của AIO.
2. **P0 — mọi ghi PCIe/MMIO trên Linux phải fail-closed và có readback**: script đang bỏ qua lỗi `setpci`, còn C injector bỏ qua lỗi fallback I/O nhưng vẫn báo `[OK]`.
3. **P1 — bỏ chính sách tắt blocklist mặc định và các tuyên bố “100% Vanguard/mọi driver”**: Microsoft và Riot hiện đều xem các lớp bảo vệ như vulnerable-driver blocklist, Secure Boot, HVCI/VBS là biện pháp bảo mật có thể được thực thi tùy cấu hình.

Các ưu tiên tiếp theo là xác minh đúng quy trình ký EFI, củng cố chuỗi cung ứng release, nâng Wails lên bản vá bảo mật, sandbox service systemd và thêm race/fuzz/coverage gates.

## Cách đọc mức độ bằng chứng

- **Đã xác minh từ tài liệu**: kết luận được nguồn sở hữu công nghệ xác nhận (Microsoft, Riot, Go, Linux kernel, systemd, GitHub, Wails, UEFI Forum, PCI-SIG/pciutils).
- **Suy luận từ code**: hành vi thấy trực tiếp trong working tree; chưa chạy trên máy thật.
- **Cần kiểm thử phần cứng**: không được xem là hoàn tất cho tới khi chạy trên CMP 30HX/40HX được hỗ trợ và ghi before/after logs.

## Ma trận ưu tiên

| Ưu tiên | Hạng mục | Lợi ích chính | Có thể kiểm thử không cần phần cứng |
|---|---|---|---|
| P0 | Chống cross-origin/CSRF cho Web UI cũ | Ngăn website khác gọi API admin trên localhost | Có, bằng `httptest` |
| P0 | Fail-closed + readback cho Linux PCIe/MMIO | Không báo thành công giả sau ghi thất bại | Có, mock `setpci` và file resource giả |
| P1 | Không tắt blocklist mặc định; lưu/khôi phục đúng trạng thái | Giảm suy yếu bảo mật và rollback sai | Có, registry abstraction/mock |
| P1 | Xác minh chữ ký EFI và quy trình enrollment | Tránh báo ký thành công nhưng firmware không thể nạp | Một phần; boot vẫn cần máy thật |
| P1 | Chuỗi cung ứng release có pin, provenance và chữ ký | Người dùng xác minh được nguồn gốc binary | Có trong CI |
| P1 | Nâng Wails 2.16.0 lên bản vá | Nhận các bản vá bảo mật upstream | Có, build/test Windows |
| P2 | Sandbox service systemd theo quyền tối thiểu | Giảm blast radius của service root | Có một phần bằng offline analysis |
| P2 | Race/fuzz/coverage gates | Bắt lỗi cạnh tranh và parser edge cases | Có trong CI |

---

## 1. P0 — Bảo vệ Web UI HTTP cũ chạy quyền Administrator

### Bằng chứng trong repo

- `windows-v3.0/tools/inst40hx/webserver.go:106-110` tự nâng quyền trước khi chạy server.
- `webserver.go:158-162` trả SSE logs với `Access-Control-Allow-Origin: *`.
- Các endpoint thay đổi trạng thái như `/api/unlock-now` (`284-297`), `/api/force-root-gen2` (`300-335`), `/api/full-install` (`338-351`) và reboot BIOS (`571-579`) không kiểm tra method, `Origin`, `Sec-Fetch-Site` hoặc token.
- `webserver.go:617-644` chỉ bind loopback; bind loopback **không tự ngăn** một trang web đang mở trong trình duyệt gửi request đến localhost.
- Cùng repo đã có mẫu tốt hơn tại `windows-v3.0/tools/aio/fallback_web.go:45-77`: chỉ chấp nhận POST cùng origin cho mutation, kiểm tra loopback host, chặn DNS rebinding/cross-origin; regression tests ở `fallback_web_security_test.go:9-88`.

### Đã xác minh từ tài liệu

Go 1.25+ cung cấp [`http.CrossOriginProtection`](https://pkg.go.dev/net/http#CrossOriginProtection), chặn non-safe cross-origin browser requests dựa trên `Sec-Fetch-Site` hoặc so sánh `Origin` với `Host`. Tài liệu cũng nhấn mạnh GET/HEAD/OPTIONS phải không gây thay đổi trạng thái. Module `inst40hx` hiện khai báo Go 1.26.5 nên có thể dùng API chuẩn này.

Wails cũng đã phải vá lớp tấn công cùng họ: changelog v2.17.0 mô tả dev server trước đó có thể bị trang độc hại kết nối đến localhost IPC và gọi bound API; bản vá thêm host allowlist, same-origin và capability cookie ([Wails v2 changelog](https://v2.wails.io/changelog/)).

### Suy luận từ code

Một trang khác có thể kích hoạt endpoint admin qua request tới `127.0.0.1`; endpoint hiện không yêu cầu nonce/cookie và mutation không bị giới hạn thành POST. Khả năng khai thác thực tế chưa được thử, nhưng điều kiện phòng vệ thiết yếu đang thiếu.

### Thay đổi tối thiểu đề xuất

- Bọc mux bằng `http.NewCrossOriginProtection().Handler(mux)`.
- Chuyển mọi mutation sang **POST-only**; trả `405` cho GET/HEAD.
- Kiểm tra `Host` là IP loopback thực, bỏ wildcard CORS và chỉ cho SSE cùng origin.
- Thêm token ngẫu nhiên theo mỗi lần khởi động (cookie `HttpOnly; SameSite=Strict` hoặc custom header) cho defense-in-depth.
- Tái sử dụng logic/test table từ `aio/fallback_web.go`, tránh tạo framework mới.
- Đặt `ReadHeaderTimeout`, `ReadTimeout`, `WriteTimeout` hợp lý cho `http.Server`; SSE cần timeout riêng hoặc context-aware.

### Acceptance criteria

- `httptest` chứng minh cross-origin, thiếu origin, DNS-rebinding host, sai method và token sai đều không gọi handler.
- Same-origin POST với token đúng hoạt động; SSE chỉ GET cùng origin.
- Không endpoint reboot/install/uninstall/unlock nào thay đổi trạng thái qua GET.
- Không cần GPU thật cho gate này.

---

## 2. P0 — Ghi PCIe/MMIO trên Linux phải fail-closed và có readback

### Bằng chứng trong repo

- `Setup_CMP30HX_LinuxAIO.sh:423,434,448` dùng `setpci ... || true` cho LNKCTL2 và MRRS; ngay sau đó `424,435,449` in `[OK]` mà không đọc lại giá trị.
- `linux-mmio/mmio_injector.c:81-91` bỏ kết quả lỗi của `fd_write_u32`/`fd_read_u32` trong nhánh fallback.
- Injector in thành công ở `mmio_injector.c:121` và trả 0 ở `125`, dù một hoặc nhiều fallback write/read có thể đã lỗi.
- `40hxcore/mrrs_readback_test.go` đã có pattern đúng phía Go: từ chối verdict thành công nếu write MRRS không được áp dụng.

### Đã xác minh từ tài liệu

Upstream [pciutils `setpci` manpage](https://github.com/pciutils/pciutils/blob/master/setpci.man) xác nhận đây là công cụ đọc/ghi configuration space, hỗ trợ masked read-modify-write và chế độ `-vD` để kiểm tra chuỗi thao tác mà chưa ghi. Linux UAPI định nghĩa Device Control tại offset `0x08`, mask MRRS `0x7000`, giá trị 512 B `0x2000`, và LNKCTL2 TLS 5.0 GT/s là `0x0002` ([Linux `pci_regs.h`](https://github.com/torvalds/linux/blob/master/include/uapi/linux/pci_regs.h)).

Linux kernel ghi rõ `resource0..N` là các tài nguyên nhị phân có thể mmap và khuyên kiểm tra kết quả mmap vì không phải nền tảng nào cũng hỗ trợ mọi resource ([PCI sysfs documentation](https://kernel.org/doc/html/latest/PCI/sysfs-pci.html)). API `write(2)` có thể trả lỗi hoặc ghi thiếu; caller phải kiểm tra số byte thực tế ([Linux man-pages `write(2)`](https://man7.org/linux/man-pages/man2/write.2.html)).

### Suy luận từ code

Hiện có đường dẫn “false success”: write thất bại nhưng log/status vẫn nói thành công. Với cấu hình PCIe, false success vừa gây chẩn đoán sai vừa có thể khiến bước sau dựa trên trạng thái chưa được áp dụng.

### Thay đổi tối thiểu đề xuất

- Bỏ `|| true` ở mọi write quan trọng; nếu cần thu stderr, lưu exit code và fail với BDF/register cụ thể.
- Sau mỗi write: đọc lại, mask đúng field, so sánh expected/actual trước khi in `[OK]`.
- Dùng masked write của `setpci` thay vì tự dựng toàn bộ word khi thích hợp, để giảm rủi ro ghi đè bit không liên quan.
- Trong C injector, biến `WR32/RD32` thành hàm trả lỗi; dừng ở lỗi đầu tiên, readback từng offset quan trọng, kiểm tra `munmap`/`close` khi hữu ích.
- Tách chuỗi TU116/TU106 theo device/profile; không suy ra hỗ trợ chỉ từ mask BOOT0 rộng.
- Giữ CMP 30HX ở Gen2; không thêm Gen3, link-disable, PnP reset hay hard fallback.

### Acceptance criteria

- Mock `setpci` trả non-zero làm script trả lỗi và status không thể là `GEN2_SUCCESS`.
- Mock “write thành công nhưng readback không đổi” cũng thất bại.
- Compile C injector trong CI với `-Wall -Wextra -Werror`; chạy trên sparse file 1 MiB đã seed BOOT0 để xác minh success/failure/readback mà không chạm hardware.
- Hardware acceptance riêng: before/after dump cho GPU + upstream port, MRRS, TLS/current speed và rollback/reboot/sleep trên CMP 30HX/40HX.

---

## 3. P1 — Không tắt vulnerable-driver blocklist theo mặc định; lưu và khôi phục đúng trạng thái

### Bằng chứng trong repo

- `windows-v3.0/tools/riotcore/registry.go:96-98` luôn ghi `VulnerableDriverBlocklistEnable=0`.
- `ConfigureDriverClassRegistry` gọi thao tác này ở `registry.go:190`, nên tối ưu đồ họa bị gắn với thay đổi chính sách bảo mật toàn máy.
- `RevertGraphicsOptimization` tại `registry.go:295-298` luôn ghi giá trị `1`, không khôi phục trạng thái/value-absence trước đó.
- README quảng bá “tương thích 100% với Riot Vanguard” (`README.md:20-21`) và hướng dẫn tắt blocklist (`97,152`).

### Đã xác minh từ tài liệu

Microsoft giải thích vulnerable-driver blocklist ngăn các driver hợp lệ nhưng có lỗ hổng bị lợi dụng để chạy mã kernel; blocklist bật mặc định từ Windows 11 2022 Update và còn được cưỡng chế khi HVCI, Smart App Control hoặc S mode hoạt động. Microsoft khuyến nghị bật HVCI hoặc dùng blocklist ([Microsoft recommended driver block rules](https://learn.microsoft.com/windows/security/threat-protection/windows-defender-application-control/microsoft-recommended-driver-block-rules)).

Riot hiện mô tả Vanguard có thể yêu cầu TPM 2.0, Secure Boot, HVCI/VBS và IOMMU tùy cấu hình ([Vanguard security requirements](https://support-leagueoflegends.riotgames.com/hc/de/articles/44545169004051-Vanguard-Sperren)); hướng dẫn VAN 9005 cũng nói Memory Integrity là một phần cấu hình Vanguard kiểm tra ([Riot VAN 9005](https://support-valorant.riotgames.com/hc/de/articles/16941220890899-Einstellungen-f%C3%BCr-Virtualisierungsbasierte-Sicherheit-VBS-auf-Windows-10-VAN9005-VALORANT)).

### Suy luận từ code

“Tối ưu game” hiện làm giảm một lớp bảo vệ kernel cho cả hệ thống, dù routing/game.cfg không cần cùng transaction đó. Hàm revert có thể thay đổi máy vốn không có value hoặc do chính sách doanh nghiệp quản lý, thay vì trả máy về đúng trạng thái ban đầu. Tuyên bố tương thích tuyệt đối không có ma trận phiên bản Windows/driver/Vanguard và còn phụ thuộc chính sách Riot thay đổi theo thời gian.

### Thay đổi tối thiểu đề xuất

- Tách routing/game.cfg khỏi blocklist. Mặc định chỉ làm thay đổi không giảm bảo mật.
- Nếu driver cụ thể bị block: hiển thị driver identity/hash và Code Integrity event, yêu cầu xác nhận riêng, không quảng bá đây là bước “an toàn”.
- Trước thay đổi, lưu atomically: key tồn tại hay không, type, value, timestamp/owner của session; chỉ rollback nếu session sở hữu thay đổi đó.
- Nếu policy do MDM/App Control/HVCI cưỡng chế, fail rõ thay vì cố ghi đè.
- Đổi README thành “đã kiểm thử với…” cùng bảng OS build/driver/game/date; bỏ “mọi driver”, “100%”, “triệt để”.

### Acceptance criteria

- Unit tests cover các trạng thái: value không tồn tại, 0, 1, access denied, policy-enforced và concurrent external change.
- Revert trả đúng trạng thái ban đầu, không mặc định ép `1`.
- CI kiểm tra README không tái xuất hiện các claim tuyệt đối nếu chưa có evidence matrix.
- Hardware/Vanguard acceptance phải ghi phiên bản cụ thể; không suy rộng sang phiên bản khác.

---

## 4. P1 — Xác minh thật sự chữ ký EFI và coi enrollment là thao tác firmware-specific

### Bằng chứng trong repo

- `windows-v3.0/tools/aio/bind_riot.go:209-220` tạo self-signed code-signing cert rồi gọi `Set-AuthenticodeSignature`, nhưng discard object kết quả.
- Chỉ cần PowerShell process exit 0, code log “ký thành công” ở `233-238`; không kiểm tra `Status`, không verify lại file và không kiểm tra firmware `db/dbx`.
- Hướng dẫn giả định BIOS có thể trực tiếp “Append db Key” từ `C:\CMP40HX_Key.cer`, trong khi giao diện/định dạng enrollment phụ thuộc firmware.

### Đã xác minh từ tài liệu

`Set-AuthenticodeSignature` chỉ thêm chữ ký cho file có Subject Interface Package hỗ trợ ([Microsoft PowerShell docs](https://learn.microsoft.com/en-us/powershell/module/microsoft.powershell.security/set-authenticodesignature?view=powershell-7.5)). UEFI 2.11 yêu cầu firmware kiểm tra chữ ký PE/COFF với authorized database `db`, đồng thời từ chối nếu hash/chứng chỉ bị `dbx` cấm; cập nhật database còn phụ thuộc user/setup mode và authenticated-variable rules ([UEFI 2.11, Secure Boot and Driver Signing](https://uefi.org/specs/UEFI/2.11/32_Secure_Boot_and_Driver_Signing.html)). Microsoft cũng nói UEFI driver phải được ký bởi key/CA trong `db` hoặc hash image phải có trong `db`; nên theo công cụ/hướng dẫn OEM cho quản lý key ([Windows Secure Boot key guidance](https://learn.microsoft.com/en-us/windows-hardware/manufacture/desktop/windows-secure-boot-key-creation-and-management-guidance?view=windows-11)).

### Suy luận từ code

Process thành công không chứng minh chữ ký có trạng thái hợp lệ, firmware hỗ trợ định dạng cert, cert đã vào đúng database, hoặc EFI image không bị dbx chặn. Đây là lý do cần đổi nhãn từ “tự động ký thành công” sang kết quả từng bước có xác minh.

### Thay đổi tối thiểu đề xuất

- Capture object `Set-AuthenticodeSignature`; yêu cầu `Status == Valid`, sau đó verify lại từ file độc lập.
- Ghi SHA-256 trước/sau, subject/thumbprint, signature status vào log; không log/export private key.
- Xuất public material theo các định dạng firmware phổ biến nhưng không tuyên bố universal; trỏ người dùng tới hướng dẫn OEM.
- Trước reboot, hiển thị rollback: cách khôi phục default Secure Boot keys và cách boot recovery media.
- Không tự ghi PK/KEK/db/dbx từ OS trong flow mặc định.

### Acceptance criteria

- Fixture EFI/PE hợp lệ và file không hỗ trợ SIP tạo hai kết quả phân biệt; unsupported/invalid không được log success.
- Có bước verify độc lập và manifest hash.
- Test boot với Secure Boot chỉ được đánh dấu “đã xác minh” sau khi chạy trên từng firmware/motherboard đại diện; release vẫn draft trước đó.

---

## 5. P1 — Củng cố chuỗi cung ứng release và binary provenance

### Bằng chứng trong repo

- `.github/workflows/ci.yml:17,81,96,133-134,157` dùng action tag mutable như `@v4`, không pin full commit SHA.
- Release job hiện build lại sáu Windows EXE và tạo checksum cho ZIP, nhưng `git archive HEAD` vẫn mang theo binary theo dõi sẵn như `linux-mmio/mmio_injector`, EFI và driver; Linux CI không compile/so sánh `mmio_injector.c`.
- Workflow chưa tạo artifact attestation; chưa verify Authenticode cho EXE release.

### Đã xác minh từ tài liệu

GitHub hiện xem workflow dependency lockfile là biện pháp bảo vệ mạnh nhất vì khóa cả dependency trực tiếp lẫn bắc cầu; nếu chưa áp dụng được lockfile, GitHub khuyến nghị pin action tới **full-length commit SHA** thay cho tag có thể bị di chuyển ([Secure use reference](https://docs.github.com/en/actions/reference/security/secure-use)). GitHub artifact attestations tạo build provenance và hỗ trợ verify nguồn gốc artifact ([Using artifact attestations](https://docs.github.com/en/actions/how-tos/secure-your-work/use-artifact-attestations/use-artifact-attestations)).

Microsoft nêu Authenticode xác minh publisher và integrity; self-signed chỉ phù hợp development/testing, còn phân phối công khai nên dùng identity tin cậy ([Code signing options](https://learn.microsoft.com/en-us/windows/apps/package-and-deploy/code-signing-options)). `SignTool verify /pa` có exit code rõ để làm CI gate ([SignTool verification](https://learn.microsoft.com/en-us/windows/win32/seccrypto/using-signtool-to-verify-a-file-signature)).

### Suy luận từ code

Checksum bên cạnh artifact trên cùng release giúp phát hiện hỏng file, nhưng không tự chứng minh workflow/repository nào đã tạo file. Tracked native binaries không được rebuild hoặc đối chiếu nguồn làm giảm khả năng audit.

### Thay đổi tối thiểu đề xuất

- Ưu tiên workflow dependency lockfile nếu repository hỗ trợ; nếu chưa, pin tất cả third-party/GitHub actions bằng full SHA, kèm comment version và dùng Dependabot/Renovate để cập nhật.
- Build `linux-mmio/mmio_injector` từ source trong Linux CI; ít nhất smoke-test trên file giả và publish hash/binary từ CI.
- Tạo SHA-256 cho từng executable/EFI/driver và ZIP; sinh provenance attestation cho release ZIP và các binary tự build.
- Khi có hạ tầng signing hợp lệ, Authenticode-sign + timestamp các EXE rồi `signtool verify /pa`; giữ khóa ở HSM/secrets service, không trong repo.
- Ghi rõ binary nào có source/reproducible build, binary nào là third-party/firmware blob, nguồn và hash của chúng.
- Giữ release ở draft cho tới hardware acceptance theo ranh giới dự án.

### Acceptance criteria

- Không còn `uses: owner/action@vN` trong workflow release.
- `gh attestation verify` xác minh được ZIP từ đúng repo/workflow/ref.
- CI build C source và chạy smoke test không hardware.
- Manifest liệt kê hash + origin của mọi EXE/SYS/EFI/native binary được phát hành.

---

## 6. P1 — Nâng Wails khỏi v2.16.0 và thêm kiểm tra advisory

### Bằng chứng trong repo

`windows-v3.0/tools/aio/go.mod:8` pin `github.com/wailsapp/wails/v2 v2.16.0`.

### Đã xác minh từ tài liệu

Upstream Wails v2.17.0 vá nhiều vấn đề bảo mật, gồm Windows notification fallback sang PowerShell và localhost dev IPC cross-origin/DNS-rebinding; v2.17.1 vá thêm Linux child-frame bridge ([Wails v2 changelog](https://v2.wails.io/changelog/)). Advisory chính chủ liệt kê lỗi Windows notification command execution là mức High ([Wails security advisories](https://github.com/wailsapp/wails/security/advisories)).

### Suy luận từ code

Dependency hiện thấp hơn bản vá. Không tìm thấy call Notification trong AIO, nên **chưa khẳng định đường khai thác notification reachable trong ứng dụng này**; tuy vậy update vẫn giảm rủi ro tương lai và nhận hardening upstream.

### Thay đổi tối thiểu đề xuất

- Nâng lên Wails v2.17.1 hoặc bản v2 mới hơn đã review; chạy `go mod tidy`, regenerate bindings nếu cần.
- Soát release notes, `govulncheck ./...` và smoke-test Windows WebView2.
- Không nhúng remote/untrusted iframe/content vào WebView.

### Acceptance criteria

- `go test ./...`, `go vet ./...`, Wails build và UI smoke test pass.
- `govulncheck` không còn advisory reachable chưa triage; mọi “not reachable” phải được ghi lại thay vì bỏ qua scanner.

---

## 7. P2 — Sandbox service systemd theo quyền tối thiểu

### Bằng chứng trong repo

`Setup_CMP30HX_LinuxAIO.sh:628-643` tạo service root `Type=oneshot` với `DefaultDependencies=no`, nhưng không có sandbox directives. Service cần PCI config/MMIO, vì vậy hardening phải đo đạc, không được bật tùy tiện các cờ có thể chặn `/sys/bus/pci`.

### Đã xác minh từ tài liệu

`systemd.exec` cung cấp `NoNewPrivileges=`, `ProtectSystem=`, capability bounding và nhiều giới hạn khác; nếu không cấu hình capability bounding thì systemd không giới hạn capability của process ([systemd.exec](https://www.freedesktop.org/software/systemd/man/latest/systemd.exec)). `systemd-analyze security` chấm exposure và hỗ trợ offline review/threshold ([systemd-analyze](https://freedesktop.org/software/systemd/man/latest/systemd-analyze.html)).

### Suy luận từ code

Service chỉ cần chạy một chuỗi PCIe cục bộ, không cần network hay ghi home. Ít nhất một số hardening như `NoNewPrivileges=yes`, `ProtectHome=yes`, `PrivateTmp=yes` có khả năng áp dụng được; capability và filesystem allowlist phải xác nhận trên distro/hardware.

### Thay đổi tối thiểu đề xuất

- Sinh unit thành fixture riêng có thể test; thêm hardening từng bước.
- Bắt đầu bằng `NoNewPrivileges=yes`, `ProtectHome=yes`, `PrivateTmp=yes`, `RestrictAddressFamilies=AF_UNIX`; đánh giá `ProtectSystem=full`.
- Chỉ thu hẹp `CapabilityBoundingSet`/`DeviceAllow` sau khi trace chính xác nhu cầu `setpci` và `resource0`.
- Thêm `systemd-analyze verify` và `systemd-analyze security --offline=yes` vào Linux CI nếu phiên bản runner hỗ trợ.

### Acceptance criteria

- Unit pass `systemd-analyze verify`.
- Có baseline và ngưỡng exposure được giải thích.
- Mock daemon vẫn chạy; hardware acceptance xác nhận PCI config/MMIO không bị sandbox chặn và suspend/resume vẫn phục hồi.

---

## 8. P2 — Race, fuzz và coverage gates cho các đường parser/concurrency

### Bằng chứng trong repo

- `.github/workflows/ci.yml:25-36` chạy `go test -cover ./...` và `go vet`, nhưng chỉ in coverage, không enforce ngưỡng.
- Chưa có `go test -race` hoặc fuzz target trong workflow.
- Dự án có concurrency đáng chú ý: SSE hub, telemetry loop và operation mutex; có nhiều parser/contract đầu vào như status, BDF/PCI state và game config.

### Đã xác minh từ tài liệu

Go mô tả data race là nhóm lỗi khó debug và cung cấp `go test -race` ([Go Race Detector](https://go.dev/doc/articles/race_detector)). Native fuzzing dùng coverage-guided inputs để tìm edge case, và corpus lỗi trở thành regression input cho `go test` thường ([Go Fuzzing](https://go.dev/doc/security/fuzz/)). Coverage profile chuẩn được tạo bằng `go test -coverprofile` và đọc qua `go tool cover -func` ([Go coverage](https://go.dev/doc/build-cover)).

### Thay đổi tối thiểu đề xuất

- Gate coverage theo module/package có logic quan trọng; bắt đầu từ baseline thực tế rồi tăng đến yêu cầu dự án >=80%, không “che” package OS glue khó mock.
- Thêm race job cho module/platform hỗ trợ; trên Windows cần toolchain C tương thích như tài liệu Go lưu ý.
- Fuzz các parser thuần: status contract, PCI/BDF/hex parsing, League config và request validation.
- Không fuzz trực tiếp phần ghi hardware; inject mock bus/filesystem.

### Acceptance criteria

- CI fail khi coverage logic cốt lõi giảm dưới threshold.
- Race job pass trên supported configuration.
- Fuzz smoke mỗi PR với thời gian ngắn; scheduled job dài hơn; corpus lỗi được commit làm regression test.
- Không test nào truy cập Registry/driver/EFI/MMIO thật.

---

## Lộ trình triển khai ít thay đổi nhất

1. **PR 1 — Web guard + Linux readback**: hai P0, test hoàn toàn mock/read-only.
2. **PR 2 — Security-state transaction + sửa README claims**: tách blocklist khỏi routing, lưu/restore exact state.
3. **PR 3 — Release hardening**: pin SHA, build C, manifest, attestation; chưa cần code-signing nếu chưa có signing identity.
4. **PR 4 — Wails update + CI quality gates**.
5. **PR 5 — EFI verification + systemd hardening**, sau đó mới mở hardware acceptance matrix.

## Những điều chưa được xác minh

- Chưa chạy trên CMP 30HX/40HX; link speed, MRRS throughput, reboot/sleep persistence và EFI boot đều **chưa xác minh**.
- Chưa xác nhận một website cross-origin đã khai thác thành công legacy Web UI; báo cáo xác nhận thiếu guard bằng code inspection.
- Chưa xác nhận Wails notification advisory reachable vì ứng dụng hiện không thấy gọi Notification API.
- Chưa xác nhận self-signed EFI hiện tại boot được trên bất kỳ firmware cụ thể nào.
- Không đánh giá tính hợp lệ của các reverse-engineered MMIO constants; cần trace nguồn và hardware logs riêng.

## Nguồn sơ cấp chính

- Microsoft: [vulnerable driver blocklist](https://learn.microsoft.com/windows/security/threat-protection/windows-defender-application-control/microsoft-recommended-driver-block-rules), [driver signing](https://learn.microsoft.com/en-us/windows-hardware/drivers/install/kernel-mode-code-signing-requirements--windows-vista-and-later-), [Secure Boot keys](https://learn.microsoft.com/en-us/windows-hardware/manufacture/desktop/windows-secure-boot-key-creation-and-management-guidance?view=windows-11), [code signing options](https://learn.microsoft.com/en-us/windows/apps/package-and-deploy/code-signing-options).
- Riot: [Vanguard security requirements](https://support-leagueoflegends.riotgames.com/hc/de/articles/44545169004051-Vanguard-Sperren), [VAN 9005/VBS](https://support-valorant.riotgames.com/hc/de/articles/16941220890899-Einstellungen-f%C3%BCr-Virtualisierungsbasierte-Sicherheit-VBS-auf-Windows-10-VAN9005-VALORANT).
- Go: [CrossOriginProtection](https://pkg.go.dev/net/http#CrossOriginProtection), [race detector](https://go.dev/doc/articles/race_detector), [fuzzing](https://go.dev/doc/security/fuzz/).
- Linux/pciutils/systemd: [PCI sysfs](https://kernel.org/doc/html/latest/PCI/sysfs-pci.html), [PCI register definitions](https://github.com/torvalds/linux/blob/master/include/uapi/linux/pci_regs.h), [setpci](https://github.com/pciutils/pciutils/blob/master/setpci.man), [systemd.exec](https://www.freedesktop.org/software/systemd/man/latest/systemd.exec).
- UEFI Forum: [UEFI 2.11 Secure Boot and Driver Signing](https://uefi.org/specs/UEFI/2.11/32_Secure_Boot_and_Driver_Signing.html).
- GitHub: [secure use of Actions](https://docs.github.com/en/actions/reference/security/secure-use), [artifact attestations](https://docs.github.com/en/actions/how-tos/secure-your-work/use-artifact-attestations/use-artifact-attestations).
- Wails: [v2 changelog](https://v2.wails.io/changelog/), [security advisories](https://github.com/wailsapp/wails/security/advisories).
