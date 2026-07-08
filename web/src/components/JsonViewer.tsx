import { useState } from 'react';

export function JsonViewer({ value, label }: { value: unknown; label?: string }) {
  const [expanded, setExpanded] = useState(false);
  let text: string;
  try {
    text = JSON.stringify(value, null, 2);
  } catch {
    text = String(value);
  }
  if (text === undefined || text === 'undefined' || text === '') {
    text = '{}';
  }
  return (
    <div>
      {label && <div className="mb-1 text-xs font-medium text-gray-600">{label}</div>}
      <pre
        className={`json-pane rounded bg-gray-50 p-2 text-xs text-gray-800 ${
          expanded ? 'max-h-[600px]' : 'max-h-40'
        }`}
      >
        <code>{text}</code>
      </pre>
      <button
        type="button"
        onClick={() => setExpanded((e) => !e)}
        className="mt-1 text-xs text-brand-700 hover:underline"
      >
        {expanded ? 'Collapse' : 'Expand'}
      </button>
    </div>
  );
}
