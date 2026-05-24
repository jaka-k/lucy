// Toggle between the visual builder and the raw JSON Schema textarea.
function lucyMode() {
  const checked = document.querySelector('input[name="schema_mode"]:checked');
  return checked ? checked.value : 'builder';
}

function lucyToggleMode() {
  const mode = lucyMode();
  const builder = document.getElementById('builder');
  const raw = document.getElementById('raw');
  if (builder) builder.classList.toggle('hidden', mode !== 'builder');
  if (raw) raw.classList.toggle('hidden', mode !== 'raw');
}

// ---- visual schema builder (recursive: object & array-of-object nest) ----

function makeFieldNode() {
  const tpl = document.getElementById('field-tpl');
  const node = tpl.content.firstElementChild.cloneNode(true);
  initFieldNode(node);
  return node;
}

function initFieldNode(node) {
  const line = node.querySelector(':scope > .field-line');
  const type = line.querySelector('.f-type');
  const itemType = line.querySelector('.f-itemtype');
  const remove = line.querySelector('.remove');
  const childrenBox = node.querySelector(':scope > .children');
  const childList = childrenBox.querySelector(':scope > .child-list');
  const addChild = childrenBox.querySelector(':scope > .add-child');

  function refresh() {
    const isArray = type.value === 'array';
    const isObject = type.value === 'object';
    itemType.classList.toggle('hidden', !isArray);
    const nests = isObject || (isArray && itemType.value === 'object');
    childrenBox.classList.toggle('hidden', !nests);
  }

  type.addEventListener('change', refresh);
  itemType.addEventListener('change', refresh);
  addChild.addEventListener('click', () => childList.appendChild(makeFieldNode()));
  remove.addEventListener('click', () => node.remove());
  refresh();
}

function serializeNode(node) {
  const line = node.querySelector(':scope > .field-line');
  const name = line.querySelector('.f-name').value.trim();
  const type = line.querySelector('.f-type').value;
  const desc = line.querySelector('.f-desc').value.trim();
  const required = line.querySelector('.f-req').value === 'yes';
  const childList = node.querySelector(':scope > .children > .child-list');

  const schema = { type: type };
  if (desc) schema.description = desc;

  if (type === 'object') {
    applyObject(schema, childList);
  } else if (type === 'array') {
    const itemType = line.querySelector('.f-itemtype').value;
    const item = { type: itemType };
    if (itemType === 'object') applyObject(item, childList);
    schema.items = item;
  }
  return { name: name, required: required, schema: schema };
}

function applyObject(target, childList) {
  const { properties, required } = serializeList(childList);
  target.properties = properties;
  if (required.length) target.required = required;
}

function serializeList(listEl) {
  const properties = {};
  const required = [];
  listEl.querySelectorAll(':scope > .field-node').forEach((n) => {
    const f = serializeNode(n);
    if (!f.name) return;
    properties[f.name] = f.schema;
    if (f.required) required.push(f.name);
  });
  return { properties: properties, required: required };
}

// Serialize the whole builder into a JSON Schema string ("" when empty).
function serializeBuilder() {
  const { properties, required } = serializeList(document.getElementById('fields'));
  if (Object.keys(properties).length === 0) return '';
  const schema = { type: 'object', properties: properties };
  if (required.length) schema.required = required;
  return JSON.stringify(schema);
}

function lucyInitBuilder() {
  const fields = document.getElementById('fields');
  const add = document.getElementById('add-field');
  if (!fields || !add) return;
  add.addEventListener('click', () => fields.appendChild(makeFieldNode()));
  fields.appendChild(makeFieldNode()); // one starter field
}

// Inject the builder's serialized schema into the htmx request at send time.
function lucyInitForm() {
  const form = document.getElementById('gen-form');
  if (!form) return;
  form.addEventListener('htmx:configRequest', (e) => {
    if (lucyMode() === 'builder') {
      e.detail.parameters['builder_schema'] = serializeBuilder();
    }
  });
}

// Download the rendered output, reading text straight from the <pre>.
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

// On releasing the Generate button, play a one-shot color shimmer on the art.
function lucyInitGenerateArt() {
  const btn = document.querySelector('#gen-form button.primary');
  const art = document.getElementById('gen-art');
  if (!btn || !art) return;
  btn.addEventListener('click', () => {
    art.classList.remove('shimmer');
    void art.offsetWidth; // force reflow so the animation can restart
    art.classList.add('shimmer');
  });
  art.addEventListener('animationend', () => art.classList.remove('shimmer'));
}

document.addEventListener('DOMContentLoaded', () => {
  lucyToggleMode();
  lucyInitBuilder();
  lucyInitForm();
  lucyInitGenerateArt();
});
