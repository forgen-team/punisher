
import { useEffect, useState } from "react";
import { AlertTriangle, ArrowLeft, History } from "lucide-react";
import { Badge, Button, SegmentedTabs, Textarea, iconButtonVariants } from "@/src/shared/ui/kit";
import { useChangePrompt } from "@/src/entities/punisher";
import { describePunisherError, type PromptBlock, type PromptHistoryEntry } from "@/src/shared/api/punisher";
import { useWideLayout } from "@/src/shared/lib/useWideLayout";
import { findUnknownVariables, withSuggestion } from "../lib/lineDiff";
import { EDITOR_TABS, type EditorTab } from "./constants";
import { DiffView } from "./DiffView";
import { HistorySheet } from "./HistorySheet";
import { PreviewView } from "./PreviewView";

interface PromptBlockEditorProps {
  block: PromptBlock;
  /** 강의 피드백에서 온 제안 문구 — 초안에만 붙이고 저장은 사용자가 한다 */
  suggestion?: string;
  /** 모바일(1단)에서 목록으로 돌아가기 */
  onBack?: () => void;
}

/** 블록 상세: 하는 일·적용 범위·경고 → 편집 / 기본값과 비교 / 미리보기 → 저장·되돌리기·이전 버전. */
export function PromptBlockEditor({ block, suggestion, onBack }: PromptBlockEditorProps) {
  const change = useChangePrompt();
  const [draft, setDraft] = useState(() => (suggestion ? withSuggestion(block.body, suggestion) : block.body));
  const [tabChoice, setTab] = useState<EditorTab>("edit");
  // 넓은 화면(xl)은 편집 옆에 '기본값과 비교'를 늘 펼쳐 두므로 비교 탭을 따로 두지 않는다.
  const wide = useWideLayout() === true;
  const tab: EditorTab = wide && tabChoice === "diff" ? "edit" : tabChoice;
  const tabs = wide ? EDITOR_TABS.filter((t) => t.value !== "diff") : EDITOR_TABS;
  const [historyOpen, setHistoryOpen] = useState(false);
  const [confirmReset, setConfirmReset] = useState(false);
  const [notice, setNotice] = useState("");

  // 다른 블록으로 바꾸거나 제안 문구가 바뀌면 초안을 새로 잡는다(저장 결과로 block.body 가 바뀔 때도).
  useEffect(() => {
    setDraft(suggestion ? withSuggestion(block.body, suggestion) : block.body);
    setConfirmReset(false);
  }, [block.id, block.body, suggestion]);

  const dirty = draft.trimEnd() !== block.body.trimEnd();
  const unknownNames = findUnknownVariables(draft, block.vars.map((v) => v.name));

  function save() {
    change.mutate(
      { kind: "save", id: block.id, body: draft },
      { onSuccess: () => setNotice("저장했어요. 다음 생성부터 적용돼요. 이미 만든 강의는 '새 프롬프트로 다시 만들기'로 새 버전을 만들 수 있어요.") },
    );
  }
  function reset() {
    change.mutate({ kind: "reset", id: block.id }, { onSuccess: () => setNotice("기본값으로 되돌렸어요. 이전 본문은 '이전 버전'에 남아 있어요.") });
    setConfirmReset(false);
  }
  function restore(entry: PromptHistoryEntry) {
    change.mutate({ kind: "restore", id: block.id, historyId: entry.id }, {
      onSuccess: () => {
        setHistoryOpen(false);
        setNotice("이전 버전으로 복원했어요.");
      },
    });
  }

  return (
    <div className="flex min-w-0 flex-col gap-4">
      <header className="flex flex-col gap-1">
        <div className="flex items-center gap-1">
          {onBack && (
            <button type="button" onClick={onBack} className={`${iconButtonVariants({ variant: "ghost", size: "sm" })} -ml-1.5`} aria-label="블록 목록으로">
              <ArrowLeft aria-hidden="true" />
            </button>
          )}
          <p className="text-xs font-semibold text-accent-text">{block.stageTitle}</p>
        </div>
        <h2 className="flex items-center gap-2 text-lg font-semibold text-fg">
          {block.title}
          {block.overridden && <Badge tone="accent">수정됨</Badge>}
        </h2>
        <p className="break-keep text-sm text-fg-secondary">{block.purpose}</p>
        <p className="break-keep text-xs text-fg-muted">적용: {block.appliesTo}</p>
      </header>

      {block.warning && (
        <p role="note" className="flex gap-2 break-keep rounded-md bg-warning-subtle px-3 py-2 text-sm text-warning">
          <AlertTriangle className="mt-0.5 size-4 shrink-0" aria-hidden="true" />
          {block.warning}
        </p>
      )}

      <SegmentedTabs aria-label="보기" items={tabs} value={tab} onChange={setTab} fill />

      {tab === "edit" && (
        <div className="grid gap-4 xl:grid-cols-2 xl:items-start">
          <div className="flex min-w-0 flex-col gap-2">
            {/* 옆 '기본값과 비교'의 안내 줄과 높이를 맞춘다 */}
            {wide && <p className="text-xs text-fg-muted">내 본문 · 고치면 오른쪽 비교가 바로 바뀌어요</p>}
            <Textarea
              aria-label={`${block.title} 본문`}
              value={draft}
              onChange={(event) => setDraft(event.target.value)}
              rows={18}
              spellCheck={false}
              className="font-mono text-xs leading-5 xl:h-[60vh]"
            />
            {block.vars.length > 0 && (
              <p className="text-xs text-fg-muted">
                쓸 수 있는 변수:{" "}
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
                이 블록이 모르는 변수는 바뀌지 않고 그대로 들어가요: {unknownNames.map((v) => `{{${v}}}`).join(", ")}
              </p>
            )}
          </div>
          {wide && <DiffView before={block.default} after={draft} />}
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
        <Button onClick={save} disabled={!dirty} loading={change.isPending && change.variables?.kind === "save"}>
          저장
        </Button>
        {dirty && (
          <Button variant="ghost" onClick={() => setDraft(block.body)}>
            편집 취소
          </Button>
        )}
        <Button variant="ghost" onClick={() => setHistoryOpen(true)}>
          <History className="size-4" aria-hidden="true" />
          이전 버전
        </Button>
        {block.overridden &&
          (confirmReset ? (
            <span className="flex items-center gap-1">
              <Button variant="secondary" onClick={reset} loading={change.isPending && change.variables?.kind === "reset"}>
                기본값으로 되돌리기 확인
              </Button>
              <Button variant="ghost" onClick={() => setConfirmReset(false)}>
                취소
              </Button>
            </span>
          ) : (
            <Button variant="ghost" onClick={() => setConfirmReset(true)}>
              기본값으로 되돌리기
            </Button>
          ))}
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
