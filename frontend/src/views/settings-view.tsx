import { SettingsForm } from "@/components/settings-form";
import { ConfigEditor } from "@/components/config-editor";

export function SettingsView() {
  return (
    <div className="mx-auto w-full max-w-5xl space-y-6 p-4">
      <div>
        <h2 className="mb-4 text-lg font-semibold text-foreground">Settings</h2>
        <SettingsForm />
      </div>
      <div>
        <h3 className="mb-4 text-sm font-semibold text-foreground">
          Advanced Configuration
        </h3>
        <ConfigEditor />
      </div>
    </div>
  );
}
