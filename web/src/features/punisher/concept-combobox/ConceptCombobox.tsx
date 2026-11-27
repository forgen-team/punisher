
import { useId, useMemo, useState, type KeyboardEvent } from "react";
import { X } from "lucide-react";
import { TextField } from "@forgen-team/ui";
import { cn } from "@/src/shared/lib/cn";
import { areaLabel, useConcepts } from "@/src/entities/punisher";
import type { Concept } from "@/src/shared/api/punisher";
import { useT } from "@/src/shared/i18n";
import { messages } from "./i18n";

const MAX_OPTIONS = 30;

function matches(concept: Concept, needle: string): boolean {
  if (!needle) { return true; }
  return concept.name.toLowerCase().includes(needle) || concept.id.toLowerCase().includes(needle);
}

interface ConceptComboboxProps {
  label: string;
  value: string[];
  onChange: (ids: string[]) => void;
  max?: number;
}

/** Searchable multi-select of concepts (ARIA combobox, ArrowUp/Down/Enter/Escape). */
export function ConceptCombobox({ label, value, onChange, max = 10 }: ConceptComboboxProps) {
  const t = useT(messages);
  const inputId = useId();
  const listId = useId();
  const { data: concepts = [], isLoading } = useConcepts();
  const [text, setText] = useState("");
  const [open, setOpen] = useState(false);
  const [active, setActive] = useState(0);

  const options = useMemo(() => {
    const needle = text.trim().toLowerCase();
    return concepts.filter((c) => !value.includes(c.id) && matches(c, needle)).slice(0, MAX_OPTIONS);
  }, [concepts, text, value]);

  const byId = useMemo(() => new Map(concepts.map((c) => [c.id, c])), [concepts]);
  const full = value.length >= max;

  function select(concept: Concept | undefined) {
    if (!concept || full) { return; }
    onChange([...value, concept.id]);
    setText("");
    setActive(0);
  }

  function onKeyDown(event: KeyboardEvent<HTMLInputElement>) {
    switch (event.key) {
      case "ArrowDown": {
        event.preventDefault();
        setOpen(true);
        setActive((index) => Math.min(index + 1, Math.max(options.length - 1, 0)));
        break;
      }
      case "ArrowUp": {
        event.preventDefault();
        setActive((index) => Math.max(index - 1, 0));
        break;
      }
      case "Enter": {
        if (open && options[active]) {
          event.preventDefault();
          select(options[active]);
        }
        break;
      }
      case "Escape": {
        if (open) { event.stopPropagation(); }
        setOpen(false);
        break;
      }
      case "Backspace": {
        if (!text && value.length > 0) { onChange(value.slice(0, -1)); }
        break;
      }
      default: {
        break;
      }
    }
  }

  const activeId = open && options[active] ? `${listId}-${options[active].id}` : undefined;

  return (
    <div className="relative flex flex-col gap-1.5">
      <label htmlFor={inputId} className="text-sm font-medium text-fg">
        {label} <span className="text-xs font-normal text-fg-muted">({value.length}/{max})</span>
      </label>
      {value.length > 0 && (
        <ul className="flex flex-wrap gap-1.5" aria-label={t.selected}>
          {value.map((id) => (
            <li key={id} className="inline-flex h-8 items-center gap-1 bg-accent-subtle pl-3 pr-1 text-sm text-accent-text">
              {byId.get(id)?.name ?? id}
              <button
                type="button"
                aria-label={t.remove(byId.get(id)?.name ?? id)}
                onClick={() => onChange(value.filter((v) => v !== id))}
                className="inline-flex size-6 items-center justify-center hover:bg-canvas-hover"
              >
                <X className="size-3.5" aria-hidden="true" />
              </button>
            </li>
          ))}
        </ul>
      )}
      <TextField
        disabled={full}
        value={text}
        placeholder={full ? t.full : isLoading ? t.loading : t.search}
        onValueChange={({ value: next }) => {
          setText(next);
          setOpen(true);
          setActive(0);
        }}
        inputProps={{
          id: inputId,
          role: "combobox",
          "aria-expanded": open && options.length > 0,
          "aria-controls": listId,
          "aria-autocomplete": "list",
          "aria-activedescendant": activeId,
          onFocus: () => setOpen(true),
          onBlur: () => setOpen(false),
          onKeyDown,
        }}
      />
      {open && options.length > 0 && (
        <ul
          id={listId}
          role="listbox"
          aria-label={t.options(label)}
          className="absolute left-0 right-0 top-full z-panel mt-1 max-h-64 overflow-y-auto rounded-md border border-line bg-canvas-elevated py-1 shadow-overlay"
        >
          {options.map((c, index) => (
            <li
              key={c.id}
              id={`${listId}-${c.id}`}
              role="option"
              aria-selected={index === active}
              onMouseDown={(event) => {
                event.preventDefault();
                select(c);
              }}
              onMouseEnter={() => setActive(index)}
              className={cn(
                "flex min-h-touch cursor-pointer items-center justify-between gap-2 px-3 text-sm md:min-h-9",
                index === active ? "bg-accent-subtle text-accent-text" : "text-fg",
              )}
            >
              <span className="truncate">{c.name}</span>
              <span className="shrink-0 text-xs text-fg-muted">{areaLabel(c.area)}</span>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
