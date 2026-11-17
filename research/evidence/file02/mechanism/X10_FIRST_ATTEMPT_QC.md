# X10 first full attempt — excluded, no inference

The first full 72-arm attempt used source `e788570532cebe18006c96cb62dab009fcfeba1b`.
It stopped in block 10 after 31 arms passed. Arm `X10-b10-I` completed
241/241 socket writes and 618,888/618,888 offered bytes but failed the
locked achieved-rate check: one steady one-second window had 17 writes
instead of 20, followed by catch-up windows of 23 and 22. Maximum send
offset jitter was about 395 ms. Its result status is `RATE_MISS`, not a
missing transaction or a DoS observation. The harness stopped before
the remaining 40 arm directories were created.

The entire private raw tree is frozen at manifest SHA-256
`b75610e83449171baca36e315b5061f6965e9a7ba31d063ac3b0cd89566d662c`.
The session's QC is `INVALID_FOR_INFERENCE`; aggregate primary/CI are
`null` and decision is `WITHHELD_QC_FAILURE`. No first-attempt block
effect was used to choose a rate, margin, n or threshold. The source's
initial JSON verifier compared Python tuples to serialized lists, causing
a derived-artifact mismatch on readback. A later verifier correction
made `verify` PASS on the unchanged raw and derived files; it did not
change QC or create an effect estimate.

The X10 plan was amended *before a second full attempt* to continue all
scheduled arms after a run-level failure, retain every failed raw run,
allow only independently verified `RATE_MISS` exclusion of its whole
block, and require at least 20 complete blocks. No failed arm is replaced.
This is a QC/completeness correction, not a post hoc effect decision.
