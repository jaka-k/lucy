// Toggle between the visual builder and the raw JSON Schema textarea.
function lucyToggleMode() {
  const checked = document.querySelector('input[name="schema_mode"]:checked');
  const mode = checked ? checked.value : 'builder';
  const builder = document.getElementById('builder');
  const raw = document.getElementById('raw');
  if (builder) builder.classList.toggle('hidden', mode !== 'builder');
  if (raw) raw.classList.toggle('hidden', mode !== 'raw');
}

// Download the rendered output as a file, reading the text straight from the
// <pre> so we don't need any server-side session state.
function downloadResult(btn) {
  const pre = document.getElementById('output');
  if (!pre) return;
  const blob = new Blob([pre.textContent], { type: btn.dataset.mime || 'text/plain' });
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = btn.dataset.filename || 'output.txt';
  document.body.appendChild(a);
  a.click();
  a.remove();
  URL.revokeObjectURL(url);
}

document.addEventListener('DOMContentLoaded', lucyToggleMode);
