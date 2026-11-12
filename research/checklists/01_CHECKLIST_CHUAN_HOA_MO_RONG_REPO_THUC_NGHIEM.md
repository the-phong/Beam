# CHECKLIST 01 — CHUẨN HÓA REPO THỰC NGHIỆM THEO THỨ TỰ BẮT BUỘC

> **Mục tiêu:** Chuẩn bị một testbed đủ đúng, đủ công bằng và đủ quan sát để **File 02** kiểm chứng lần lượt **RQ1 (PQC/hybrid có tạo tác động tài nguyên khác biệt?) → RQ2 (có tín hiệu bất thường phát hiện được đủ sớm?) → RQ3 (ML có hơn cơ chế đơn giản?)**.
>
> **Quy tắc thực hiện:** Chỉ làm **chặng hiện tại**. Một chặng chỉ được đánh dấu `OK` khi **tất cả task + bằng chứng + Gate** của chặng đó đạt. Chưa `OK` thì sửa ngay trong chặng đó, **không mở chặng sau**. Không lấy các phase cũ hay các kết quả sẵn có trong repo làm tiêu chuẩn khoa học.
>
> **Repo đầu vào lịch sử:** `the-phong/DoS-in-Blockchain-PQC` (nay là `the-phong/Beam`); giữ lại các thành phần đã đúng, sửa/thay thế phần không đạt. **Không xóa hoặc ghi đè chứng cứ lịch sử.** Toàn bộ workload gây tải chỉ chạy trong testbed cô lập và có cơ chế dừng an toàn.

> **Checkpoint refactor 2026-11-12 (lịch sử):** GitHub repository đã đổi tên thành [Beam](https://github.com/the-phong/Beam); ứng dụng và Go module vẫn là Pluto. Owner đã merge [PR #18](https://github.com/the-phong/Beam/pull/18) vào `test-mainet` tại `e8f20726f5cea94b47c4dcddb756e39d21207448`. [CI sau merge](https://github.com/the-phong/Beam/actions/runs/43146676567) trên đúng commit này đã **SUCCESS**. Controller **CONDITIONAL ACCEPT** cho structural refactor; [hồ sơ review](../evidence/refactor/CONTROLLER_REVIEW_CLOSEOUT.md) ghi giới hạn và việc quản trị còn mở. `master` vẫn là default branch cũ; PR #13 sang `master` đang xung đột, chưa merge. **S07.1–S07.5 vẫn chưa làm/chưa tích; S07 chưa được Owner mở và File 02 chưa được phép chạy.** Các checkpoint bên dưới là lịch sử tại thời điểm ghi.

> **Tiến độ S07 2026-11-12:** Owner đã merge [PR #19](https://github.com/the-phong/Beam/pull/19) vào `test-mainet` tại `49facc28ca29102e3e7295546f29b74fe3503a00` và giao S07 trên nhánh `research/s07-reproducibility` trong folder gốc. Beam chỉ là tên repo; ứng dụng/module vẫn Pluto. S07.1–S07.5 có [báo cáo kỹ thuật](../evidence/s07/RUN_READINESS_REPORT.md), session smoke C/CP/P/H, manifest SHA-256 và quick CI/secret scan/full regression đều PASS trên [Draft PR #20](https://github.com/the-phong/Beam/pull/20) tại commit `24edcb4`. Gate S07 **OK kỹ thuật trong phạm vi smoke**; quick CI, secret scan và full regression đều PASS trên commit tài liệu `9efeb8b`; chờ Controller/Owner nghiệm thu; File 02 vẫn đóng. Dòng trạng thái phía trên là checkpoint trước S07.

> **Cập nhật kiểm kê 2026-11-09 (lịch sử):** Checklist này là bản điều hành hiện hành; các dấu tích từ V1 không được tự động chuyển sang. S01 đạt 4/4 task, Gate S01: OK; S02 đạt 5/5 mục soạn thảo, Gate S02: OK theo xác nhận Owner sau merge PR #12. S03 đạt 4/4 mục và đã merge PR #14; S04 đạt 4/4 mục với Gate kỹ thuật cục bộ OK ở thời điểm ghi. S01 cố định baseline lịch sử `0bdd869f2c6b3ed59b0ae825954d8667b3c7481e`; S04 xuất phát từ `test-mainet` merge commit `1c8084207adb5da787e34e230dd0dc6d9a3de05f` trên nhánh `research/s04-bft-correctness`, chỉ trong folder gốc `pluto`. Trạng thái refactor cũ giữ trong stash cục bộ `7ec52bd`. Bằng chứng S01: [BASELINE_AUDIT](../evidence/s01/BASELINE_AUDIT.md), [PIPELINE_MAP](../evidence/s01/PIPELINE_MAP.md), [hash manifest](../evidence/s01/baseline_manifest.json), [B0/B1 inventory](../evidence/s01/LEGACY_INVENTORY.md), [security audit](../evidence/s01/SECURITY_AUDIT.md). Bundle sao lưu và log gốc chỉ lưu cục bộ trong `pluto/research/artifacts/private-baseline/` (được Git ignore), vì chứa lịch sử hoặc dữ liệu chạy thử chưa duyệt để công bố.

> **Tiến độ 2026-11-10:** Owner đã merge [PR #15](https://github.com/the-phong/DoS-in-Blockchain-PQC/pull/15) vào `test-mainet` tại `630f2d8c81398f84073796747108c30ab9dee3c8` và giao S05. Nhánh hiện hành `research/s05-observability` trong đúng folder `pluto`. S05 có bằng chứng kỹ thuật cục bộ cho 5/5 mục, gồm số giao dịch/byte P2P nhận trực tiếp và thành phần hàng đợi socket CheckTx từ bản CometBFT v0.38.26 instrument riêng cho S05. Gate kỹ thuật cục bộ **OK**, chờ review độc lập và Owner; S06–S07 chưa mở.

> **Tiến độ 2026-11-11:** Owner đã merge [PR #16](https://github.com/the-phong/DoS-in-Blockchain-PQC/pull/16) vào `test-mainet` tại `ac91250ddff31d07b6dcb19f81878e149256e84f` và giao S06. Nhánh `research/s06-safe-testbed` ở đúng folder `pluto` có 5/5 mục S06 được kiểm chứng bằng corpus, unit/race và pilot bảy lượt trên bốn validator thật cùng một host; [report](../evidence/s06/TESTBED_REPORT.md) ghi kết quả/giới hạn. Gate kỹ thuật cục bộ OK, chờ review độc lập và Owner; S07 chưa mở. Các dòng trên là checkpoint lịch sử tại thời điểm ghi.

> **Sắp xếp thư mục 2026-11-10:** Chỉ còn một cây `research/`. Checklist, protocol, corpus, evidence và artifacts đã được chuyển vào đây; hồ sơ M01 được đặt cùng S01. Đường dẫn bằng chứng đã cập nhật. Không thay đổi dấu tích hoặc trạng thái Gate; S04 vẫn chờ Controller/Owner review, S05 chưa mở.

> **Tên thư mục mã 2026-11-10:** C/CP/P/H dùng `internal/profiles/` và `cmd/profile-*`; mã corpus/benchmark B0/B1 dùng `internal/benchmarks/`, `cmd/b0b1-corpus/`, `research/benchmarks/b0b1/` và `research/corpus/b0b1/`. Không còn thư mục mã hiện hành đặt theo `v2` hoặc `phase05`; các định danh format trên wire và hồ sơ chứng cứ lịch sử giữ nguyên. `make verify`, race tests, S04 bốn validator, hai corpus verifier và hai B0/B1 smoke đều exit 0 sau khi đổi đường dẫn. Không thay đổi dấu tích hoặc Gate.

> **Điều hướng source sau PR #17:** Owner đã merge S06 vào `test-mainet` tại `ed6bc8f474c13ed709d5ab5f57b3856edcbf602a` và giao refactor kiến trúc riêng. Những đường dẫn source trong checkpoint lịch sử ở dưới là cách viết tại thời điểm kiểm chứng; dùng [bảng ánh xạ](../../docs/REFACTOR_MAPPING.md) để chạy lại trên cây mới. Refactor này không đổi dấu tích hoặc mở S07.

## S01 — Đóng băng bản gốc và kiểm kê đường xử lý [BẮT BUỘC ĐẦU TIÊN]

- [x] **S01.1** Ghi commit/tree nguồn, nhánh, version Go/CometBFT/cryptolib, cấu hình, lệnh build/test và tạo bản sao lưu + hash manifest. **Đã kiểm chứng:** `BASELINE_AUDIT.md`; bundle Git đầy đủ 4.084.006 byte, `git bundle verify` đạt, SHA-256 `0310c9738e001c11e010bb69c7d85fcb5a2a1f3933ab684a4680d746c2f6fd42`; manifest 148 hash SHA-256 `2aafada7f216984e9c8cb75bdbc41f9734bbfc58aa1679d7188f08e180a644de`.
- [x] **S01.2** Chạy lại build, test, smoke B0/B1 và 4-validator; tách kết quả **tái lập được** khỏi **chỉ có trong báo cáo**. **Đã kiểm chứng:** `make verify` exit 0; `make test-integration` đạt 13/13; B0 bốn validator đạt 7/7; B1 bốn validator đạt 4/4; P2P relay smoke đạt. `BASELINE_AUDIT.md` ghi lệnh, exit code, SHA-256 log, các lượt lỗi và cách sửa; `LEGACY_INVENTORY.md` tách kết quả tái lập khỏi báo cáo lịch sử.
- [x] **S01.3** Đọc code và vẽ đường đi **RPC/P2P → precheck → CheckTx → crypto verify → mempool → relay → proposal/commit**; ghi vị trí reject/cache/recheck. **Đã kiểm chứng:** `PIPELINE_MAP.md` ghi đường đi và dòng mã CometBFT v0.38.26 cùng ứng dụng B0/B1; test peer relay dưới quorum đạt. Các phép đo tải/relay trực tiếp vẫn thuộc chặng sau.
- [x] **S01.4** Ghi rõ các khác biệt hiện có giữa B0/B1: wire size, khóa trên wire/genesis, registry, state, persistence, verify order; kiểm tra secret/log riêng tư. **Đã kiểm chứng:** `LEGACY_INVENTORY.md` liệt kê delta gây nhiễu so sánh; `SECURITY_AUDIT.md` ghi quét Gitleaks baseline và worktree, phân loại false positive ở nhánh khác, giữ raw log riêng tư ngoài Git.

**Bàn giao:** `BASELINE_AUDIT.md`, `PIPELINE_MAP.md`, snapshot/hash manifest và log build/test.

**GATE S01 — OK:** Có bằng chứng đọc code + chạy thử, có thể truy vết lại chính xác bản gốc; mọi khác biệt B0/B1 ảnh hưởng so sánh đã được liệt kê.

**Trạng thái Gate S01: OK (kiểm kê kỹ thuật và chức năng trên localhost).** Lượt cuối: `make verify` exit 0, log SHA-256 `a16fe85d47ef4a914a1a8b680e487fc107aeb7aa5c59f5f04ec7656b509b18bb`; `make test-integration` exit 0 sau 246,189 giây, 13/13 bài, log `5850ab59b9725e46ad0ece56e492d372304f0dfe10466438d04d308aff412e97`; B0 bốn validator exit 0, 7/7 bài, log `7ac7fba88e6fd64464945364b778f05f9094a3469cb23da08aa10c3bd0bf5b15`; B1 bốn validator exit 0, 4/4 bài, log `cdc9a500b209116dde8436b9bcfa97fc6cb309761b935d4cb35331c9039917ad`; P2P relay smoke exit 0, log `cbbbd1070fe69a986ed2cbad5b1b6ccba56817999ca1129058a0e16ff1a1c825`. Manifest 148 mục được tái tạo byte-đúng; `git bundle verify` đạt. Lỗi M6 đã tái hiện 1/10 trước sửa, do test gửi SIGTERM trước khi Comet cài signal trap; test harness sửa xong đạt 10/10. Timeout integration cũ 240 giây đã tăng mặc định lên 900 giây, không đổi assertion. Tất cả lượt thất bại, gồm lượt thử điều kiện AppHash sai đã loại bỏ, được ghi trong `BASELINE_AUDIT.md` cùng hash log. **Giới hạn:** các bài bốn validator chạy trên một máy WSL2, chưa chứng minh kết quả tài nguyên/mạng nhiều host. S02 có thể bắt đầu; chưa tích bất kỳ mục S02 nào.

---

## S02 — Khóa bài toán và hợp đồng thực nghiệm [CHỈ SAU S01]

- [x] **S02.1** Chốt đúng ba RQ: **RQ1 Resource Impact**, **RQ2 Detectability**, **RQ3 Baseline Gap**; ghi H0/H1 và điều kiện bác bỏ từng RQ. **Bằng chứng soạn thảo:** [Research protocol](../protocol/RESEARCH_PROTOCOL.md#research-questions-and-falsifiable-decisions); ngưỡng đề xuất chờ duyệt độc lập.
- [x] **S02.2** Chốt phạm vi phân biệt: **pre-admission verification exhaustion**, **accepted-mempool byte/count pressure**, **P2P relay congestion**; không gọi mọi rejection là “tràn mempool”. **Bằng chứng soạn thảo:** [Threat model](../protocol/THREAT_MODEL.md#mechanisms-that-must-remain-separate).
- [x] **S02.3** Chốt threat model A/B/C: người gửi tùy ý, người sửa giao dịch đã ký, chủ khóa hợp lệ; định nghĩa nhóm tải hợp lệ cao và lỗi client là **benign** khi phù hợp. **Bằng chứng soạn thảo:** [Actor capabilities và workload labels](../protocol/THREAT_MODEL.md#actor-capabilities).
- [x] **S02.4** Chốt bốn cấu hình đối chứng **C (ECDSA)**, **CP (ECDSA + padding hợp lệ)**, **P (ML-DSA-44)**, **H (ECDSA + ML-DSA-44)**; cùng intent, nghiệp vụ, trạng thái và điều kiện tải. **Bằng chứng soạn thảo:** [Four controlled profiles](../protocol/RESEARCH_PROTOCOL.md#four-controlled-authentication-profiles); tính tương đương cần được thử ở S03.
- [x] **S02.5** Chốt biến kiểm soát, chỉ số chính (legitimate goodput, latency, CPU, mempool bytes, relay bytes), định nghĩa thời điểm cảnh báo và quy tắc không rò rỉ thông tin tương lai; định nghĩa tiêu chí dừng an toàn. **Bằng chứng soạn thảo:** [Measurement contract](../protocol/MEASUREMENT_CONTRACT.md); telemetry cần được cài và thử ở S05–S07.

**Bàn giao:** `RESEARCH_PROTOCOL.md`, `THREAT_MODEL.md`, `MEASUREMENT_CONTRACT.md`.

**GATE S02 — OK:** Người khác đọc tài liệu có thể nói rõ **đo cái gì, đo tại đâu, dùng đối chứng nào, thế nào là tấn công/quá tải** mà không cần suy đoán.

**Trạng thái Gate S02: OK (Owner xác nhận 2026-11-09).** PR #12 đã merge vào `test-mainet` tại `2a3dcbceb52b996425d911dde87b691317f56efd`; Owner xác nhận S02 đã được xem xét và cho phép tiếp tục S03. GitHub PR không có review/comment riêng, nên nguồn quyết định gate là lời xác nhận của Owner trong phiên làm việc, không phải một review GitHub được suy diễn. S03 bắt đầu trên nhánh riêng từ merge commit này.

---

## S03 — Xây bốn profile so sánh C / CP / P / H [CHỈ SAU S02]

- [x] **S03.1** Chuẩn hóa **một đường xử lý nghiệp vụ/state/consensus chung**; chỉ thay đổi yếu tố crypto và kích thước theo thiết kế đối chứng, tránh lấy toàn bộ B0/B1 delta làm “chi phí ML-DSA”. **Bằng chứng:** [PROFILE_EQUIVALENCE](../protocol/PROFILE_EQUIVALENCE.md), `internal/profiles/app/app.go`, `TestOneABCIPathForFourProfiles`; cùng genesis, transfer, AppHash và callbacks. App trực tiếp trong bộ nhớ; chạy node thật thuộc S04.
- [x] **S03.2** Hoàn thiện **C, CP, P, H**; CP phải là giao dịch **được chấp nhận hợp lệ**, có kích thước khớp H theo chính sách định trước (không bị reject vì padding). **Bằng chứng:** [public vectors](../../internal/experiment/profiles/testdata/public_vectors.json), `TestFourProfilesSameTransferAndExactSize`, `TestCPAndHExactBytesAcrossChainIDBounds`; CP/H 2.565 byte với chain ID vector, cả hai được chấp nhận.
- [x] **S03.3** Chuẩn hóa canonical signed intent, nonce/balance, khóa và binding, thứ tự xác minh/dừng sớm; định nghĩa chính sách public key on-wire hoặc genesis và giữ biến này được kiểm soát. **Bằng chứng:** [PROFILE_EQUIVALENCE](../protocol/PROFILE_EQUIVALENCE.md#canonical-wire-and-authentication), `internal/profiles/{tx,state}.go`; khóa ở genesis, profile tách domain, secp→ML→state, từ chối không đổi state.
- [x] **S03.4** Viết test từng profile: valid, malformed, wrong chain, wrong signature, wrong binding, wrong nonce, duplicate; đo chính xác kích thước wire và đường xác minh đạt tới. **Bằng chứng:** [PROFILE_AUDIT](../evidence/s03/PROFILE_AUDIT.md), `internal/profiles/{profile,vector}_test.go`, `internal/profiles/app/app_test.go`; `make verify` và race test exit 0.

**Bàn giao:** Source bốn profile, `PROFILE_EQUIVALENCE.md`, golden vectors và unit tests.

**GATE S03 — OK:** Bốn profile làm cùng một tác vụ chuyển tiền; **C vs CP** cô lập kích thước, **CP vs H** kiểm soát kích thước để khảo sát phần chi phí xử lý còn lại; tất cả test profile đạt.

**Trạng thái Gate S03: OK ở phạm vi source và direct-ABCI.** Bốn profile có cùng trạng thái sau transfer; CP là giao dịch hợp lệ và dài bằng H; unit/direct-ABCI/race tests đạt. [Báo cáo đối chứng](../protocol/PROFILE_EQUIVALENCE.md) và [audit](../evidence/s03/PROFILE_AUDIT.md) ghi lệnh, kết quả, hash và giới hạn. Owner đã merge PR #14 vào `test-mainet` tại `1c8084207adb5da787e34e230dd0dc6d9a3de05f` và giao S04 ngày 2026-11-09.

---

## S04 — Xác minh tính đúng trên mạng BFT [CHỈ SAU S03]

- [x] **S04.1** Tạo corpus kiểm tra cho cả bốn profile; mỗi case có `case_id`, bytes/hash, expected code, stage, actor capability và điều kiện tiên quyết. **Bằng chứng:** [corpus công khai](../corpus/s04_public_corpus.json) 34 case, SHA-256 `d0f4bfaea4c541ee6f71547e44c30aef588704051459e57e04d6478c83`; generator/validator `cmd/profile-corpus`, oracle `TestS04CorpusABCIOracle`.
- [x] **S04.2** Kiểm thử trực tiếp `CheckTx`, `ProcessProposal`, `FinalizeBlock` và đường commit; xác nhận lỗi không làm biến đổi state. **Bằng chứng:** [CORRECTNESS_REPORT](../evidence/s04/CORRECTNESS_REPORT.md), `TestS04CorpusABCIOracle`, `TestS04ProposalDuplicateAndRecheck`; rejected input không đổi balance/nonce/supply, block height vẫn tăng khi finalize.
- [x] **S04.3** Chạy bốn validator trên testbed cô lập; kiểm tra các giao dịch hợp lệ được commit đúng một lần và block/AppHash/nonce/balance khớp ở cùng height. **Bằng chứng:** [log bốn validator đã lọc](../evidence/s04/four_validator_summary.log), `TestS04FourValidatorProfiles`; C/CP/P/H đều PASS, bốn danh tính tham gia, AppHash hậu transfer giống nhau tại H3. Bốn validator cùng một host WSL2.
- [x] **S04.4** Kiểm tra cache/dedup/recheck và xác thực việc lỗi sớm không đi tới phép verify đắt đỏ; recorder tắt/bật không đổi quyết định xác minh. **Bằng chứng:** Comet cache chặn replay qua RPC; oracle thử dedup trong proposal/FinalizeBlock và gọi `CheckTxType_Recheck` trực tiếp; recorder ON/OFF cùng code/state cho 34 case, reject sớm không vào secp/ML. Comet-driven recheck với tx còn trong mempool chưa được cưỡng bức riêng; giới hạn ghi trong [report](../evidence/s04/CORRECTNESS_REPORT.md).

**Bàn giao:** `CORRECTNESS_REPORT.md`, corpus + oracle + log 4-validator.

**GATE S04 — OK:** Không có divergence hoặc state mutation do input bị từ chối; các lớp workload thực sự đi qua stage dự kiến. **Chưa đúng thì không đo hiệu năng.**

**Trạng thái Gate S04: OK kỹ thuật cục bộ, chờ Controller/Owner review.** `make verify`, race test và `make test-s04-4v` exit 0; [CORRECTNESS_REPORT](../evidence/s04/CORRECTNESS_REPORT.md) ghi lệnh, hash, 34 oracle case, kết quả bốn profile và giới hạn. S05 chưa được giao, không chạy phép đo tải hay tuyên bố DoS.

**Đóng S04 sau merge:** Owner merge PR #15 vào `test-mainet` tại `630f2d8c81398f84073796747108c30ab9dee3c8` ngày 2026-11-10 và giao S05. Câu trạng thái phía trên là checkpoint trước merge.

**Checkpoint S04 2026-11-09:** source commit `b8ccd7cf8c500fe2495143a21c76606d41c558d2`, tree `0c3c331e2929e576a2d5bf133be3624ca107087c`, trên `research/s04-bft-correctness` trong folder gốc `pluto`; [Draft PR #15](https://github.com/the-phong/DoS-in-Blockchain-PQC/pull/15) nhắm `test-mainet`. `make verify` sau commit exit 0, log SHA-256 `0d64b569c197abdb5a80a0a274976649e6ab47e62e1f480b131a9ee131790a94`. Controller/Owner chưa chấp nhận hoặc merge PR #15.

---

## S05 — Bổ sung telemetry đúng vị trí [CHỈ SAU S04]

- [x] **S05.1** Instrument riêng: ingress (tx/s, bytes/s, RPC/P2P), precheck/reject reason, CheckTx queue/service, ECDSA verify, ML-DSA verify và state check. **Bằng chứng:** proxy đo RPC; `Reactor.Receive` instrument trực tiếp tổng giao dịch và payload byte P2P; socket client đo chờ từ lúc gửi vào hàng đợi đến lúc writer lấy ra; app đo service/secp/ML/state. Rate lấy delta hai mẫu cùng PID trên thời gian monotonic, không trộn RPC với P2P. [Từ điển](../protocol/METRICS_DICTIONARY.md) nêu rõ phạm vi và `NOT MEASURED` cho tổng enqueue→app/propagation.
- [x] **S05.2** Đo mempool **accepted count + accepted bytes**, từ chối trước mempool, duplicate/cache; không cộng giao dịch invalid bị `CheckTx` loại vào occupancy. **Bằng chứng:** native `num_unconfirmed_txs` count/bytes, proxy phân loại `app_reject`/`duplicate_cache`; test mạng thật xác nhận giao dịch lỗi không vào occupancy, fixture kiểm tra count 2/bytes 2575. [Report](../evidence/s05/TELEMETRY_REPORT.md) ghi giới hạn chưa bắt được đỉnh occupancy dương trên mạng thật.
- [x] **S05.3** Đo P2P relay (ingress/egress bytes, latency/queue khi quan sát được); nếu chưa instrument được thì gắn nhãn `NOT MEASURED`, không suy diễn từ RPC. **Bằng chứng:** Comet Prometheus `p2p_peer_{send,receive}_bytes_total` kênh mempool `0x30` có giá trị dương trong test bốn validator; propagation latency và hàng đợi riêng kênh là `NOT MEASURED`, không lấy từ RPC.
- [x] **S05.4** Đo per-process CPU/RSS cùng profiler cho crypto hotspot; tách client RPC latency, thời gian chờ và thời gian xử lý; đồng hồ monotonic cho duration. **Bằng chứng:** sampler CPU/RSS từng PID, CPU profile, client RPC/proxy/app service riêng; Comet socket queue đo trực tiếp bằng monotonic. Test bốn profile có queue count 5 mỗi lượt và tổng duration dương. Không gọi thành phần queue này là toàn bộ thời gian chờ đến app.
- [x] **S05.5** Kiểm thử metrics ON/OFF: cùng quyết định/state; đo overhead instrumentation, dropped samples, schema/timestamp và bảo vệ bí mật. **Bằng chứng:** 34 case New/Recheck ON/OFF và hai lượt mạng bốn profile có cùng AppHash; benchmark 5 lần CP/H, test drop/schema/localhost/privacy đạt. [Report](../evidence/s05/TELEMETRY_REPORT.md) ghi hash log, kết quả và nhiễu phép đo.

**Bàn giao:** `METRICS_DICTIONARY.md`, collector/structured logs, kiểm thử ON/OFF.

**GATE S05 — OK:** Mỗi chỉ số có **định nghĩa, vị trí, đơn vị, thời điểm có sẵn và chất lượng phép đo**; không nhầm CPU process với CPU riêng PQC, RPC với relay.

**Trạng thái Gate S05: kỹ thuật cục bộ OK (5/5 mục có bằng chứng), chờ review độc lập và Owner chấp nhận.** [METRICS_DICTIONARY](../protocol/METRICS_DICTIONARY.md) ghi định nghĩa/đơn vị/điểm đo/thời điểm có sẵn/chất lượng từng chỉ số; [TELEMETRY_REPORT](../evidence/s05/TELEMETRY_REPORT.md) ghi test thật và giới hạn. Tổng thời gian enqueue→app, P2P propagation và queue relay theo kênh vẫn `NOT MEASURED`; không suy diễn từ số đo khác. S06 chưa được phép bắt đầu.

---

## S06 — Chuẩn hóa testbed và bộ tạo tải an toàn [CHỈ SAU S05]

- [x] **S06.1** Pin phiên bản/cấu hình/mempool limits/consensus; tách load generator và collector khỏi tiến trình validator; ghi rõ nếu chỉ chạy **4 validator cùng một host** (chỉ đủ cho pilot, không tuyên bố độc lập hạ tầng). **Bằng chứng:** [TESTBED_CONFIG](../protocol/TESTBED_CONFIG.md), [config máy đọc được](../config/s06_pilot.json); test tạo bốn tiến trình Comet, bốn collector riêng và một `profile-load` riêng; pilot một host WSL2, không tuyên bố độc lập hạ tầng.
- [x] **S06.2** Xây open-loop generator có `planned/actual send timestamps`, unique tx, nonce/account management và phân biệt nguồn RPC/P2P. **Bằng chứng:** `internal/profiles/load/{plan,runner,pilot}.go`, `cmd/profile-load`, `TestS06IsolatedOpenLoopPilot`; plan mỗi tx ký sẵn, hash duy nhất, mỗi sender nonce 0 riêng; thời điểm gửi thực là HTTP socket-write callback. Chỉ nguồn RPC loopback được phép; P2P relay đo riêng, không giả làm P2P injection.
- [x] **S06.3** Hỗ trợ ba kiểu tải **same tx/s**, **same bytes/s**, **fixed legitimate tx/s + added invalid tx/s**; nếu không đạt offered rate thì run phải báo lỗi. **Bằng chứng:** ba regime trong `plan.go`, `TestS06PilotPlansHaveDirectOracleAndMatchedRegimes`, `TestS06OpenLoopQueueMissIsFailure`; [pilot summary](../evidence/s06/pilot_summary.log) xác nhận 7/7 lượt đạt planned/actual, C/CP cùng target byte, P/H cùng target tx, H fixed legitimate+invalid; `RATE_MISS` khi queue/rate không đạt.
- [x] **S06.4** Chuẩn hóa workload: benign low/high/burst, invalid early, invalid ECDSA, invalid ML-DSA sau ECDSA hợp lệ, late state-invalid, valid high-rate; mỗi vector có khả năng attacker tạo ra được. **Bằng chứng:** [corpus S06](../corpus/s06_scenarios.json) SHA-256 `6f43b725156f60deb0f0d2e6c451605bae479a5e890ff58f7e3be46c5a8ca0e2`, `make corpus-s06-verify` dùng oracle code/stage/capability; pilot thực có H valid 6 tx/s, burst và H bad-ML sau secp hợp lệ. Corpus là ví dụ chuẩn hóa, chưa phải mẫu thống kê.
- [x] **S06.5** Thêm warm-up/steady-state/drain, randomize thứ tự run, giới hạn CPU/RAM/queue/time, auto-abort/kill switch; các thử nghiệm gây tải chỉ lên hệ thống riêng. **Bằng chứng:** seed 611 xáo bảy lượt, 1/4/3 giây; config hard cap 25 tx/s, 65536 byte/s, outstanding 16, 60 giây, CPU/RAM/disk; `TestS06RuntimeGuardAbortsOutstandingLoad`, `TestS06KillSwitchAndPublicTargetRefused` và [report](../evidence/s06/TESTBED_REPORT.md). Không thử đẩy host tới ngưỡng vật lý.

**Bàn giao:** `TESTBED_CONFIG.md`, load generator, corpus có nhãn/kịch bản, safety test.

**GATE S06 — OK:** Bộ tạo tải đạt mức gửi yêu cầu trên mạng kiểm soát, oracle xác nhận workload đúng loại, giới hạn an toàn hoạt động; không có tấn công ra public network.

**Trạng thái Gate S06: kỹ thuật cục bộ OK, chờ review độc lập và Owner chấp nhận.** `make test-s06-pilot` exit 0: bảy mạng bốn validator cô lập, 135/135 lần gửi theo kế hoạch, 131 giao dịch hợp lệ commit và 4 invalid được reject đúng oracle; [TESTBED_REPORT](../evidence/s06/TESTBED_REPORT.md) ghi hash log, cấu hình, seed và giới hạn. Đây là pilot cùng một host; calibration benign, precision/FPR 1%, clock nhiều host và kết quả RQ1–RQ3 **chưa có** theo [measurement contract](../protocol/MEASUREMENT_CONTRACT.md). S07 chưa được Owner giao.

---

## S07 — Tái lập, QA và bàn giao sang File 02 [CHỈ SAU S06]

- [x] **S07.1** Tạo lệnh orchestrator chạy `setup → warm-up → execute → drain → oracle → collect → analyze → freeze`; mọi run có ID, seed, commit, config hash, profile, host, offered rates. **Bằng chứng:** [`make test-s07-smoke`](../../Makefile), [orchestrator](../tools/s07_orchestrate.py), [REPRODUCE](../../REPRODUCE.md) và manifest session `s07-20261112T061020Z-6a222cf3` trong [readiness report](../evidence/s07/RUN_READINESS_REPORT.md).
- [x] **S07.2** Raw logs/results **bất biến**; có SHA-256, schema check, run-exclusion ledger; run lỗi được đánh dấu và rerun có ID mới, không ghi đè. **Bằng chứng:** `verify` kiểm lại 92 file đóng băng, unit tamper/O_EXCL đạt; bốn session lỗi giữ `INVALID_FOR_ANALYSIS` riêng và run thành công có ID mới, [hash/ledger](../evidence/s07/RUN_READINESS_REPORT.md#exclusion-ledger-and-failure-handling).
- [x] **S07.3** Analyzer tối thiểu tạo bảng size/CPU/latency/queue/mempool/relay/goodput theo **từng run**, nêu số mẫu và khoảng tin cậy; chưa kết luận DoS khi chưa đo tới điểm bão hòa. **Bằng chứng:** [sample công khai](../evidence/s07/smoke_sample.json) và [report](../evidence/s07/RUN_READINESS_REPORT.md); bốn row đủ trường, coverage relay được ghi, độ trễ inclusion là cận trên và goodput quan sát là cận dưới. CI 95% để `null` vì mỗi profile chỉ có một run độc lập; không suy diễn DoS.
- [x] **S07.4** CI nhanh: build, unit, golden vectors, schema, no-secret và 4-validator smoke; full experiment chạy riêng trên testbed cô lập. **Bằng chứng:** [quick workflow](../../.github/workflows/s07-quick.yml) [PASS](https://github.com/the-phong/Beam/actions/runs/43184028485) trên commit `9efeb8b`, gồm bốn validator C/CP/P/H; [secret scan PASS](https://github.com/the-phong/Beam/actions/runs/43184028517) trên cùng PR. Full confirmatory experiments không chạy trong CI.
- [x] **S07.5** Chạy **smoke end-to-end** đủ C/CP/P/H và vài case benign/invalid nhỏ; có `REPRODUCE.md`, `RUN_READINESS_REPORT.md` và một bộ output mẫu **được ghi rõ là smoke, không dùng làm kết luận khoa học**. **Bằng chứng:** [REPRODUCE](../../REPRODUCE.md), [readiness report](../evidence/s07/RUN_READINESS_REPORT.md), [sample](../evidence/s07/smoke_sample.json) nhãn `SMOKE_ONLY_NOT_SCIENTIFIC`; bốn profile mỗi profile 9/9 send, 5 valid commit, 4 invalid reject, hosted smoke PASS.

**Bàn giao:** Orchestrator + analyzer + CI + `REPRODUCE.md` + `RUN_READINESS_REPORT.md`.

**GATE S07 — OK:** Người khác chạy lại một smoke có thể truy vết **source → config → workload → raw → derived → oracle**, kết quả hợp lệ; từ đây **mới mở File 02**.

**Trạng thái Gate S07: OK kỹ thuật trong phạm vi smoke.** Năm task có bằng chứng; quick CI, secret scan và hai full regression đều PASS trên commit `9efeb8b`. Review độc lập và Owner nghiệm thu vẫn là gate Git; không tự mở File 02. S07 smoke không tạo kết luận RQ1–RQ3.

---

## Lệnh điều hành duy nhất: Không nhảy bước

| Thứ tự | Chặng | Được bắt đầu khi |
|---|---|---|
| 1 | S01 — Audit | Ngay lập tức |
| 2 | S02 — Research contract | S01 = OK |
| 3 | S03 — Four profiles | S02 = OK |
| 4 | S04 — Correctness BFT | S03 = OK |
| 5 | S05 — Observability | S04 = OK |
| 6 | S06 — Loadgen & testbed | S05 = OK |
| 7 | S07 — Reproduce & QA | S06 = OK |
| 8 | **File 02: T → R → A → Q → Gate RQ1 → D (RQ2, RQ3)** | S07 = OK; RQ2 chỉ mở nếu RQ1 được đánh giá đủ chất lượng; RQ3 chỉ mở sau RQ2 |

**Không làm ở File 01:** huấn luyện ML, dựng dashboard phức tạp, thêm GNN/LLM/FL, làm mitigation/admission policy mới, benchmark phá tải toàn phần, tự tuyên bố đã phát hiện DoS. Những việc này chỉ mở khi **File 02** xác lập nhu cầu thực tế.

**Tối giản được phép:** dùng lại code đúng của repo; JSON/CSV có schema thay vì dashboard; giữ một-host cho **smoke/CPU pilot**, chỉ yêu cầu multi-host khi cần kết luận network/relay mạnh. Không được tối giản **C/CP/P/H, correctness, ground truth, telemetry timestamps, safety và dữ liệu tái lập**.

## Phiếu nghiệm thu gọn cho mỗi chặng

```text
Chặng: S0X
Commit / config hash:
Task hoàn thành: .../...
Bằng chứng: [đường dẫn log, test, report, SHA-256]
Gate: OK / REWORK
Giới hạn chưa kiểm chứng:
Chặng tiếp theo được mở: Có / Không
```

> **Nguyên tắc:** Task đã có trong repo cũ chỉ được đánh dấu OK sau khi kiểm chứng lại theo chuẩn này. Không sửa File 02 để hợp thức hóa kết quả sẵn có.
