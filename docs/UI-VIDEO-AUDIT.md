# Video-based desktop UI qualification — November 12, 2026

## Video coverage
- Input: \`Screencast from 2026-11-12 21-12-56.webm\`
- Native resolution: **1906 × 877**
- Duration: **64.871653 seconds**
- All **929** decoded frames reviewed in order, indexed **0000–0928**.
- Evidence: 26 sequential contact sheets (6×6; last sheet contains 29 frames), every-frame timestamp/change CSV, and original-resolution review of major UI transitions. Full audit bundle is supplied with the review in ChatGPT; do not claim a video only sampled once per second.

## Findings grounded in the recording
- **05.42–23.22 s (frames 71–207):** Adding an appliance puts \`[object PointerEvent]\` into the NAS address input, because the original click handler passes a browser event to an address argument. Sign-in form is disproportionately tall with the warning dominating the flow.
- **23.37–31.22 s (frames 208–311):** The Overview repeats a NAS address, status and the console handoff in several bordered panels. The counters are extremely wide and low in information value.
- **33.37–35.99 s (frames 367–426):** Opening the actual Zero1Local console sends the user into a separate browser, whose device dashboard is more informative than the companion's summary. The browser-origin security boundary is legitimate; the UI handoff still needs to be clearer.
- **37.41–38.51 s (frames 464–481):** One saved appliance is presented as a narrow row in a largely empty page; device identity and status are understated.
- **38.54–42.63 s (frames 482–547):** File Manager visibly renders literal \`null\`, then simultaneously displays loading and initial load actions. Share list has weak selection hierarchy.
- **42.66–55.40 s (frames 548–781):** Shares/Access opens the manual server/path/protocol connection form before the verified NAS export inventory; this reverses the natural task.
- **46.17–48.87 s and 55.43–56.48 s:** Mapped Drives shows an appropriate empty state, but the screencast does not establish successful mounting.
- **56.48–58.45 s (frames 803–819):** Storage shows unformatted usage objects and raw mdstat in the summary area; capacity and RAID health need dedicated presentation.
- **58.45–64.87 s (frames 820–928):** System shows \`null\`, unstructured/partly loaded content, and the many management links make navigation overly dense.

## Source remediation in Beta 5 development
1. Fix the accidental PointerEvent value and add frontend regression coverage.
2. Add a persistent current-NAS dropdown and administrator session indication in the toolbar.
3. Place appliance selection before workspace navigation; collapse secondary management functions.
4. Make the dashboard NAS-first and task-first with a single primary identity/status area, compact mapping summary and contextual actions.
5. Present verified share inventory before the manual SMB/NFS form.
6. Render capacity gauges, bytes, RAID inventory and an opt-in raw mdstat disclosure.
7. Fix literal \`null\` and simultaneous Load/Loading states.
8. Reduce card/decorative chrome and align with original Zero1Local palette, first-party logos and typography.
9. Preserve the NAS security boundary, authenticated APIs, local CSRF, the update workflow and existing file/mount actions.

## Not yet qualified
This visual audit and frontend source change cannot independently verify actual Windows/Linux WebView runtime behavior, real NAS permissions, SMB/NFS mount success, or the end-to-end GitHub updater. Preserve the Beta 4 release unchanged until Beta 5 builds and acceptance gates pass.
