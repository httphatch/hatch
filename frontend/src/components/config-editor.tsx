import { useCallback, useEffect, useRef, useState } from "react";
import { EditorView, basicSetup } from "codemirror";
import { yaml } from "@codemirror/lang-yaml";
import { EditorState } from "@codemirror/state";
import { Button } from "@/components/ui/button";
import { useConfig } from "@/hooks/use-config";

const theme = EditorView.theme({
  "&": {
    fontSize: "13px",
    border: "1px solid var(--color-border)",
    borderRadius: "0.5rem",
  },
  "&.cm-focused": {
    outline: "none",
  },
  ".cm-scroller": {
    fontFamily: "ui-monospace, SFMono-Regular, Menlo, monospace",
  },
  ".cm-gutters": {
    backgroundColor: "var(--color-secondary)",
    borderRight: "1px solid var(--color-border)",
  },
});

export function ConfigEditor() {
  const { yaml: initialYaml, loading, saving, error, save } = useConfig();
  const editorRef = useRef<HTMLDivElement>(null);
  const viewRef = useRef<EditorView | null>(null);
  const [saveError, setSaveError] = useState<string | null>(null);
  const [saved, setSaved] = useState(false);

  useEffect(() => {
    if (loading || !editorRef.current) return;
    if (viewRef.current) return;

    const state = EditorState.create({
      doc: initialYaml,
      extensions: [basicSetup, yaml(), theme],
    });

    viewRef.current = new EditorView({
      state,
      parent: editorRef.current,
    });

    return () => {
      viewRef.current?.destroy();
      viewRef.current = null;
    };
  }, [loading, initialYaml]);

  const handleSave = useCallback(async () => {
    if (!viewRef.current) return;
    const content = viewRef.current.state.doc.toString();
    setSaveError(null);
    setSaved(false);
    try {
      await save(content);
      setSaved(true);
      setTimeout(() => setSaved(false), 2000);
    } catch (err) {
      setSaveError(
        err instanceof Error ? err.message : "Failed to save config"
      );
    }
  }, [save]);

  if (loading) {
    return <p className="py-8 text-center text-text-muted">Loading config…</p>;
  }

  return (
    <div className="space-y-3">
      <div ref={editorRef} className="overflow-hidden rounded-lg" />
      {(error || saveError) && (
        <p className="text-sm text-destructive">{saveError || error}</p>
      )}
      {saved && (
        <p className="text-sm text-emerald-600">
          Config saved — daemon reloaded.
        </p>
      )}
      <div className="flex justify-end">
        <Button onClick={handleSave} disabled={saving}>
          {saving ? "Saving…" : "Save & Reload"}
        </Button>
      </div>
    </div>
  );
}
