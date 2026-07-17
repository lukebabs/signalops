import { optionsRatioQualityStyle } from '../lib/optionsQuality';

// Reusable G132 options ratio-quality badge. usable -> success; partial_zero ->
// warning; all_zero / denominator_zero / empty / missing -> blocked/error;
// unknown -> muted. Optionally renders a custom label (defaults to the token).
// Read-only presentation only.
export function OptionsQualityBadge({ quality, label }: { quality: string; label?: string }) {
  return (
    <span
      className={`inline-flex shrink-0 items-center whitespace-nowrap rounded border px-1.5 py-0.5 text-[11px] font-medium ${optionsRatioQualityStyle(quality)}`}
    >
      {label ?? quality ?? '—'}
    </span>
  );
}
