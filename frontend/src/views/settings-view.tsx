import { SettingsForm } from "@/components/settings-form";
import { ConfigEditor } from "@/components/config-editor";

export function SettingsView() {
  return (
    <div className="mx-auto w-full max-w-5xl">
      <div className="flex items-center border-b border-border bg-surface/50 px-4 py-2">
        <h2 className="text-lg font-semibold text-foreground">Settings</h2>
      </div>
      <div className="space-y-6 p-4">
        <SettingsForm />
        <div>
          <h3 className="mb-4 text-sm font-semibold text-foreground">
            Advanced Configuration
          </h3>
          <ConfigEditor />
        </div>
      </div>
    </div>
  );
}
