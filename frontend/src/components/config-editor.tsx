import { useCallback, useEffect, useRef, useState } from "react";
import { toast } from "sonner";
import { EditorView, basicSetup } from "codemirror";
import { yaml } from "@codemirror/lang-yaml";
import { oneDark } from "@codemirror/theme-one-dark";
import { EditorState } from "@codemirror/state";
import { Button } from "@/components/ui/button";
import { LoadingState } from "@/components/status-messages";
import { useConfig } from "@/hooks/use-config";

const overrides = EditorView.theme({
  "&": {
    fontSize: "13px",
    border: "1px solid var(--color-border)",
  },
  "&.cm-focused": {
    outline: "none",
  },
  ".cm-scroller": {
    fontFamily: "ui-monospace, SFMono-Regular, Menlo, monospace",
  },
});

export function ConfigEditor() {
  const { yaml: initialYaml, loading, saving, error, save } = useConfig();
  const editorRef = useRef<HTMLDivElement>(null);
  const viewRef = useRef<EditorView | null>(null);
  const [saveError, setSaveError] = useState<string | null>(null);

  useEffect(() => {
    if (loading || !editorRef.current) return;
    if (viewRef.current) return;

    const state = EditorState.create({
      doc: initialYaml,
      extensions: [basicSetup, yaml(), oneDark, overrides],
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
    try {
      await save(content);
      toast.success("Config saved. Daemon reloaded.");
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : "Failed to save config"
      );
    }
  }, [save]);

  if (loading) {
    return <LoadingState message="Loading config..." className="py-8" />;
  }

  return (
    <div className="space-y-3">
      <div ref={editorRef} className="overflow-hidden" />
      {(error || saveError) && (
        <p className="text-sm text-destructive">{saveError || error}</p>
      )}
      <div className="flex justify-end">
        <Button onClick={handleSave} disabled={saving}>
          {saving ? "Saving..." : "Save & Reload"}
        </Button>
      </div>
    </div>
  );
}
