import { SettingsForm } from "@/components/settings-form";
import { ConfigEditor } from "@/components/config-editor";

export function SettingsView() {
  return (
    <div className="mx-auto w-full max-w-5xl">
      <div className="flex items-center border-b border-border bg-surface/50 px-4 py-2">
        <h2 className="text-lg font-semibold text-foreground">Settings</h2>
      </div>
      <SettingsForm />
      <div className="border-b border-border">
        <div className="flex items-center border-b border-border bg-surface/50 px-4 py-2">
          <h3 className="text-sm font-semibold text-foreground">Advanced Configuration</h3>
        </div>
        <div className="px-4 py-4">
          <ConfigEditor />
        </div>
      </div>
    </div>
  );
}
