
import { useEffect, useRef, useState } from "react";
import { AlertTriangle, History, Info } from "lucide-react";
import { Badge, Button, IconButton, SegmentedControl, SegmentedControlItem, TextField, useConfirm } from "@forgen-team/ui";
import { useDraftState } from "@/src/shared/lib/drafts";
import { useLeaveGuard } from "@/src/shared/lib/leave-guard";
import { useChangePrompt } from "@/src/entities/punisher";
import { describePunisherError, type PromptBlock, type PromptHistoryEntry } from "@/src/shared/api/punisher";
import { ReadingOverlay, toOverlayAnchor, type OverlayAnchor } from "@/src/shared/ui/punisher";
import { findUnknownVariables, withSuggestion } from "../lib/lineDiff";
import { useT } from "@/src/shared/i18n";
import { EDITOR_TABS, type EditorTab } from "./constants";
import { messages } from "../i18n";
import { DiffView } from "./DiffView";
import { HistorySheet } from "./HistorySheet";
import { PreviewView } from "./PreviewView";

interface PromptBlockEditorProps {
  block: PromptBlock;
  /** 강의 피드백에서 온 제안 문구 — 초안에만 붙이고 저장은 사용자가 한다 */
  suggestion?: string;
  /** 화면 제목이 이미 블록 이름이면(폰 전체 화면) 머리의 제목 글자는 스크린리더용으로만 둔다 */
  titleInFrame?: boolean;
}

/**
 * 블록 상세: 하는 일·경고 → 편집 / 기본값과 비교 / 미리보기 → 저장·되돌리기·이전 버전.
 * 적용 범위는 제목 옆 정보 버튼의 팝오버(모바일은 시트)에 둔다(헤더는 제목만). 모바일 뒤로가기는 셸 상단 바가 맡는다.
 */
export function PromptBlockEditor({ block, suggestion, titleInFrame = false }: PromptBlockEditorProps) {
  const change = useChangePrompt();
  const confirm = useConfirm();
  const t = useT(messages);
  const tabLabels: Record<EditorTab, string> = { edit: t.tabEdit, diff: t.tabDiff, preview: t.tabPreview };
  // 초안은 블록·저장본(body)마다 이 기기에 보관한다 — 다른 블록·화면에 갔다 와도, 새로고침해도 이어서 쓴다(점검 5).
  const startText = suggestion ? withSuggestion(block.body, suggestion) : block.body;
  const draftState = useDraftState(`prompt:${block.id}`, startText, block.body, (value) => value.trimEnd() === block.body.trimEnd());
  const draft = draftState.value;
  const setDraft = draftState.setValue;
  // 목록 옆 편집 칸은 폭이 좁아 '기본값과 비교'는 늘 탭으로 둔다(옆에 펼치지 않는다).
  const [tab, setTab] = useState<EditorTab>("edit");
  const tabs = EDITOR_TABS.map((value) => ({ value, label: tabLabels[value] }));
  const [historyOpen, setHistoryOpen] = useState(false);
  const [notice, setNotice] = useState("");
  const [scopeAnchor, setScopeAnchor] = useState<OverlayAnchor | null>(null);

  // 제안 문구가 바뀌면(강의 피드백에서 다시 들어옴) 초안에 새로 붙인다. 보관된 초안이 있으면 그대로 둔다.
  const seenSuggestion = useRef(suggestion);
  const restored = draftState.restored;
  useEffect(() => {
    if (seenSuggestion.current === suggestion) { return; }
    seenSuggestion.current = suggestion;
    if (suggestion && !restored) { setDraft(withSuggestion(block.body, suggestion)); }
  }, [suggestion, restored, block.body, setDraft]);

  const dirty = draft.trimEnd() !== block.body.trimEnd();
  // 저장하지 않은 편집이 있으면 앱 안 이동 전에 묻는다(초안은 이미 보관돼 있다).
  useLeaveGuard(dirty);
  const unknownNames = findUnknownVariables(draft, block.vars.map((v) => v.name));

  function save() {
    change.mutate(
      { kind: "save", id: block.id, body: draft },
      { onSuccess: () => { draftState.clear(); setNotice(t.savedNotice); } },
    );
  }
  async function reset() {
    const ok = await confirm({ title: t.resetConfirmTitle(block.title), description: t.resetConfirmBody, confirmLabel: t.reset, cancelLabel: t.cancel, tone: "neutral" });
    if (!ok) { return; }
    change.mutate({ kind: "reset", id: block.id }, { onSuccess: () => { draftState.clear(); setNotice(t.resetNotice); } });
  }
  function restore(entry: PromptHistoryEntry) {
    change.mutate({ kind: "restore", id: block.id, historyId: entry.id }, {
      onSuccess: () => {
        setHistoryOpen(false);
        setNotice(t.restoredNotice);
      },
    });
  }

  return (
    <div className="flex min-w-0 flex-col gap-4">
      <header className="flex flex-col gap-1">
        <p className="text-xs font-semibold text-accent-text">{block.stageTitle}</p>
        <h2 className="flex items-center gap-2 text-xl font-semibold text-fg">
          <span className={titleInFrame ? "sr-only" : undefined}>{block.title}</span>
          {block.overridden && <Badge tone="brand" variant="weak">{t.modified}</Badge>}
          <IconButton
            size="sm"
            variant="ghost"
            aria-label={t.appliesToTitle}
            aria-haspopup="dialog"
            aria-expanded={scopeAnchor !== null}
            onClick={(event) => setScopeAnchor(toOverlayAnchor(event.currentTarget.getBoundingClientRect()))}
            icon={<Info aria-hidden="true" />}
          />
        </h2>
        <ReadingOverlay open={scopeAnchor !== null} onClose={() => setScopeAnchor(null)} anchor={scopeAnchor} title={t.appliesToTitle}>
          <p className="break-keep text-sm text-fg">{t.appliesTo(block.appliesTo)}</p>
        </ReadingOverlay>
        <p className="break-keep text-sm text-fg-secondary">{block.purpose}</p>
      </header>

      {block.warning && (
        <p role="note" className="flex gap-2 break-keep rounded-md bg-warning-subtle px-3 py-2 text-sm text-warning">
          <AlertTriangle className="mt-0.5 size-4 shrink-0" aria-hidden="true" />
          {block.warning}
        </p>
      )}

      <SegmentedControl aria-label={t.tabsLabel} value={tab} onValueChange={(v) => setTab(v as EditorTab)}>
        {tabs.map((item) => (
          <SegmentedControlItem key={item.value} value={item.value}>{item.label}</SegmentedControlItem>
        ))}
      </SegmentedControl>

      {tab === "edit" && (
        <div className="flex flex-col gap-4">
          <div className="flex min-w-0 flex-col gap-2">
            <TextField
              multiline
              value={draft}
              onValueChange={({ value }) => setDraft(value)}
              textareaProps={{
                "aria-label": t.bodyLabel(block.title),
                rows: 18,
                spellCheck: false,
                className: "font-mono text-xs leading-5",
              }}
            />
            {draftState.restored && dirty && (
              <p className="flex flex-wrap items-center gap-2 text-xs text-fg-secondary">
                {t.draftRestored}
                <Button size="sm" variant="ghost" onClick={draftState.discard}>{t.draftDiscard}</Button>
              </p>
            )}
            {block.vars.length > 0 && (
              <p className="text-xs text-fg-muted">
                {t.variables}{" "}
                {block.vars.map((v, index) => (
                  <span key={v.name}>
                    {index > 0 && " · "}
                    <code className="text-fg-secondary">{`{{${v.name}}}`}</code> {v.description}
                  </span>
                ))}
              </p>
            )}
            {unknownNames.length > 0 && (
              <p role="alert" className="text-xs text-warning">
                {t.unknownVarsEdit(unknownNames.map((v) => `{{${v}}}`).join(", "))}
              </p>
            )}
          </div>
        </div>
      )}
      {tab === "diff" && <DiffView before={block.default} after={draft} />}
      {tab === "preview" && <PreviewView blockId={block.id} draft={draft} />}

      <div aria-live="polite" className="empty:hidden">
        {change.error ? (
          <p role="alert" className="text-sm text-danger">{describePunisherError(change.error)}</p>
        ) : notice ? (
          <p className="text-sm text-fg-secondary">{notice}</p>
        ) : null}
      </div>

      <div className="flex flex-wrap items-center gap-2 border-t border-line pt-3">
        <Button variant="primary" onClick={save} disabled={!dirty} loading={change.isPending && change.variables?.kind === "save"}>
          {t.save}
        </Button>
        {dirty && (
          <Button variant="ghost" onClick={() => setDraft(block.body)}>
            {t.cancelEdit}
          </Button>
        )}
        <Button variant="ghost" onClick={() => setHistoryOpen(true)}>
          <History className="size-4" aria-hidden="true" />
          {t.history}
        </Button>
        {block.overridden && (
          <Button variant="ghost" onClick={() => void reset()} loading={change.isPending && change.variables?.kind === "reset"}>
            {t.reset}
          </Button>
        )}
      </div>

      <HistorySheet
        blockId={block.id}
        open={historyOpen}
        restoring={change.isPending && change.variables?.kind === "restore"}
        onClose={() => setHistoryOpen(false)}
        onRestore={restore}
        onLoad={(entry) => {
          setDraft(entry.body);
          setTab("diff");
          setHistoryOpen(false);
        }}
      />
    </div>
  );
}
