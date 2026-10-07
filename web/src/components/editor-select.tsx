import React from 'react';
export const editors: Record<string, string> = {
  vscode: 'VS Code',
  zed: 'Zed',
  trae: 'Trae',
  cursor: 'Cursor',
};
export function EditorSelect({
  inherit = false,
  ...props
}: React.SelectHTMLAttributes<HTMLSelectElement> & { inherit?: boolean }) {
  return (
    <select {...props}>
      {inherit && <option value="">跟随默认 IDE</option>}
      {Object.entries(editors).map(([key, label]) => (
        <option key={key} value={key}>
          {label}
        </option>
      ))}
    </select>
  );
}
