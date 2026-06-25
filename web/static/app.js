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

// Inject builder schema and mongo fields into the htmx request at send time.
function lucyInitForm() {
  const form = document.getElementById('gen-form');
  if (!form) return;
  form.addEventListener('htmx:configRequest', (e) => {
    if (lucyMode() === 'builder') {
      e.detail.parameters['builder_schema'] = serializeBuilder();
    }

    const sel = document.getElementById('collection-select');
    const newName = document.getElementById('new-collection-name');
    const tag = document.getElementById('tag');
    const autoCommit = document.getElementById('auto-commit');

    if (sel && sel.value && sel.value !== '__new__') {
      e.detail.parameters['collection_id'] = sel.value;
    } else if (sel && sel.value === '__new__' && newName && newName.value.trim()) {
      e.detail.parameters['new_collection'] = newName.value.trim();
    }
    if (tag && tag.value.trim()) {
      e.detail.parameters['tag'] = tag.value.trim();
    }
    if (autoCommit) {
      e.detail.parameters['auto_commit'] = autoCommit.checked ? '1' : '0';
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

// ---- MongoDB collection picker ----

function lucyCollectionChanged(sel) {
  const newWrap = document.getElementById('new-collection-wrap');
  if (!newWrap) return;
  newWrap.classList.toggle('hidden', sel.value !== '__new__');
  if (sel.value && sel.value !== '__new__') {
    lucyLoadSchema(sel.value);
    lucyShowCollectionView(sel);
    lucyReloadItems();
  } else {
    lucyHideCollectionView();
  }
}

async function lucyLoadSchema(collectionId) {
  try {
    const res = await fetch(`/collections/${collectionId}/schema`);
    if (res.status === 204) return;
    if (!res.ok) return;
    const schema = await res.json();
    lucyDeserializeBuilder(schema);
  } catch (_) {}
}

// Rebuild the visual builder from a stored item schema.
function lucyDeserializeBuilder(schema) {
  const fields = document.getElementById('fields');
  if (!fields || schema.type !== 'object' || !schema.properties) return;
  fields.innerHTML = '';
  for (const [name, propSchema] of Object.entries(schema.properties)) {
    const required = Array.isArray(schema.required) && schema.required.includes(name);
    fields.appendChild(makeFieldNodeFromSchema(name, propSchema, required));
  }
  // Switch to builder mode
  const builderRadio = document.querySelector('input[name="schema_mode"][value="builder"]');
  if (builderRadio) { builderRadio.checked = true; lucyToggleMode(); }
}

function makeFieldNodeFromSchema(name, schema, required) {
  const node = makeFieldNode();
  const line = node.querySelector(':scope > .field-line');
  line.querySelector('.f-name').value = name;
  if (schema.description) line.querySelector('.f-desc').value = schema.description;
  line.querySelector('.f-req').value = required ? 'yes' : 'no';

  const typeEl = line.querySelector('.f-type');
  const itemTypeEl = line.querySelector('.f-itemtype');
  typeEl.value = schema.type || 'string';
  typeEl.dispatchEvent(new Event('change'));

  if (schema.type === 'array' && schema.items) {
    itemTypeEl.value = schema.items.type || 'string';
    itemTypeEl.dispatchEvent(new Event('change'));
    if (schema.items.type === 'object' && schema.items.properties) {
      const childList = node.querySelector(':scope > .children > .child-list');
      for (const [cName, cSchema] of Object.entries(schema.items.properties)) {
        const cRequired = Array.isArray(schema.items.required) && schema.items.required.includes(cName);
        childList.appendChild(makeFieldNodeFromSchema(cName, cSchema, cRequired));
      }
    }
  } else if (schema.type === 'object' && schema.properties) {
    const childList = node.querySelector(':scope > .children > .child-list');
    for (const [cName, cSchema] of Object.entries(schema.properties)) {
      const cRequired = Array.isArray(schema.required) && schema.required.includes(cName);
      childList.appendChild(makeFieldNodeFromSchema(cName, cSchema, cRequired));
    }
  }
  return node;
}

// ---- preview / selective commit modal ----

function lucyOpenCommitModal() {
  const pending = document.getElementById('pending-items');
  const pre = document.getElementById('output');
  if (!pending || !pre) return;

  let items;
  try { items = JSON.parse(pre.textContent); } catch (_) { return; }
  if (!Array.isArray(items)) return;

  const modal = document.getElementById('commit-modal');
  const hint = document.getElementById('modal-hint');
  const list = document.getElementById('modal-item-list');
  const status = document.getElementById('modal-status');

  const collectionName = pending.dataset.collectionName;
  const tag = pending.dataset.tag;
  hint.textContent = `Select items to commit to "${collectionName}"${tag ? ' / ' + tag : ''}.`;
  status.textContent = '';
  list.innerHTML = '';

  items.forEach((item, i) => {
    const label = document.createElement('label');
    label.className = 'modal-item';
    const cb = document.createElement('input');
    cb.type = 'checkbox';
    cb.checked = true;
    cb.dataset.index = String(i);
    const pre = document.createElement('pre');
    pre.className = 'modal-item-pre';
    pre.textContent = JSON.stringify(item, null, 2);
    label.appendChild(cb);
    label.appendChild(pre);
    list.appendChild(label);
  });

  modal.classList.remove('hidden');
}

function lucyCloseCommitModal() {
  const modal = document.getElementById('commit-modal');
  if (modal) modal.classList.add('hidden');
}

async function lucyCommit() {
  const pending = document.getElementById('pending-items');
  const pre = document.getElementById('output');
  const status = document.getElementById('modal-status');
  if (!pending || !pre) return;

  let items;
  try { items = JSON.parse(pre.textContent); } catch (_) { return; }
  if (!Array.isArray(items)) return;

  const checked = document.querySelectorAll('#modal-item-list input[type=checkbox]:checked');
  const selected = Array.from(checked).map(cb => items[parseInt(cb.dataset.index)]);

  status.textContent = '// committing…';
  try {
    const res = await fetch('/commit', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        collection_id: pending.dataset.collectionId,
        collection_name: pending.dataset.collectionName,
        tag: pending.dataset.tag,
        items: selected,
      }),
    });
    if (!res.ok) { status.textContent = 'Error: ' + (await res.text()); return; }
    const data = await res.json();
    status.textContent = `// ${data.inserted} inserted`;
    // update the pending badge to reflect commit
    pending.innerHTML = `&#x25B6; ${data.inserted} committed &rarr; ${pending.dataset.collectionName}${pending.dataset.tag ? ' / ' + pending.dataset.tag : ''}`;
    pending.classList.remove('pending');
    setTimeout(lucyCloseCommitModal, 900);
  } catch (e) {
    status.textContent = 'Error: ' + e.message;
  }
}

// ---- collection items browser ----

// Cache of the latest items array for the visible collection, indexed by _id.
let lucyItemsById = {};
let lucyCurrentCollection = { id: '', name: '' };

function lucyShowCollectionView(sel) {
  const view = document.getElementById('collection-view');
  const name = document.getElementById('cv-name');
  if (!view) return;
  const opt = sel.options[sel.selectedIndex];
  lucyCurrentCollection = { id: sel.value, name: opt ? opt.text : '' };
  if (name) name.textContent = '// ' + lucyCurrentCollection.name;
  view.classList.remove('hidden');
}

function lucyHideCollectionView() {
  const view = document.getElementById('collection-view');
  if (view) view.classList.add('hidden');
  lucyCurrentCollection = { id: '', name: '' };
  lucyItemsById = {};
}

async function lucyReloadItems() {
  if (!lucyCurrentCollection.id) return;
  const list = document.getElementById('cv-list');
  const hint = document.getElementById('cv-hint');
  if (!list || !hint) return;

  hint.textContent = '// loading…';
  list.innerHTML = '';
  lucyItemsById = {};
  lucySetActionsEnabled(false);

  try {
    const res = await fetch(`/collections/${lucyCurrentCollection.id}/items`);
    if (!res.ok) { hint.textContent = 'Error: ' + (await res.text()); return; }
    const items = await res.json();
    if (!items.length) {
      hint.textContent = '// no documents yet';
      return;
    }
    hint.textContent = `// ${items.length} document${items.length === 1 ? '' : 's'} (latest first)`;
    items.forEach((doc) => {
      lucyItemsById[doc._id] = doc;
      list.appendChild(lucyRenderItemRow(doc));
    });
  } catch (e) {
    hint.textContent = 'Error: ' + e.message;
  }
}

function lucyRenderItemRow(doc) {
  const label = document.createElement('label');
  label.className = 'item-row';

  const radio = document.createElement('input');
  radio.type = 'radio';
  radio.name = 'cv-selected';
  radio.value = doc._id;
  radio.addEventListener('change', () => lucySetActionsEnabled(true));

  const body = document.createElement('div');
  body.className = 'item-row-body';

  const meta = document.createElement('div');
  meta.className = 'item-row-meta';
  const idSpan = document.createElement('span');
  idSpan.className = 'item-row-id';
  idSpan.textContent = doc._id;
  meta.appendChild(idSpan);
  if (doc.tag) {
    const tagSpan = document.createElement('span');
    tagSpan.className = 'item-row-tag';
    tagSpan.textContent = doc.tag;
    meta.appendChild(tagSpan);
  }
  if (doc.created_at) {
    const ts = document.createElement('span');
    ts.className = 'item-row-ts';
    ts.textContent = String(doc.created_at).replace('T', ' ').replace('Z', '');
    meta.appendChild(ts);
  }

  const pre = document.createElement('pre');
  pre.className = 'item-row-pre';
  pre.textContent = JSON.stringify(lucyVisibleFields(doc), null, 2);

  body.appendChild(meta);
  body.appendChild(pre);
  label.appendChild(radio);
  label.appendChild(body);
  return label;
}

// Strip server-managed fields for the preview pane.
function lucyVisibleFields(doc) {
  const out = {};
  for (const k of Object.keys(doc)) {
    if (k === '_id' || k === 'created_at') continue;
    out[k] = doc[k];
  }
  return out;
}

function lucySetActionsEnabled(on) {
  const edit = document.getElementById('cv-edit');
  const del = document.getElementById('cv-delete');
  if (edit) edit.disabled = !on;
  if (del) del.disabled = !on;
}

function lucySelectedItem() {
  const checked = document.querySelector('input[name="cv-selected"]:checked');
  if (!checked) return null;
  return lucyItemsById[checked.value] || null;
}

// ---- delete modal ----

function lucyOpenDeleteModal() {
  const item = lucySelectedItem();
  if (!item) return;
  document.getElementById('confirm-preview').textContent =
    JSON.stringify(lucyVisibleFields(item), null, 2);
  document.getElementById('confirm-status').textContent = '';
  document.getElementById('confirm-modal').classList.remove('hidden');
}

function lucyCloseConfirmModal() {
  document.getElementById('confirm-modal').classList.add('hidden');
}

async function lucyConfirmDelete() {
  const item = lucySelectedItem();
  const status = document.getElementById('confirm-status');
  if (!item) return;
  status.textContent = '// deleting…';
  try {
    const res = await fetch(
      `/collections/${lucyCurrentCollection.id}/items/${item._id}`,
      { method: 'DELETE' }
    );
    if (!res.ok && res.status !== 204) {
      status.textContent = 'Error: ' + (await res.text());
      return;
    }
    status.textContent = '// deleted';
    setTimeout(() => {
      lucyCloseConfirmModal();
      lucyReloadItems();
    }, 600);
  } catch (e) {
    status.textContent = 'Error: ' + e.message;
  }
}

// ---- edit modal ----

function lucyOpenEditModal() {
  const item = lucySelectedItem();
  if (!item) return;
  document.getElementById('edit-json').value =
    JSON.stringify(lucyVisibleFields(item), null, 2);
  document.getElementById('edit-status').textContent = '';
  document.getElementById('edit-modal').classList.remove('hidden');
}

function lucyCloseEditModal() {
  document.getElementById('edit-modal').classList.add('hidden');
}

async function lucyConfirmEdit() {
  const item = lucySelectedItem();
  const status = document.getElementById('edit-status');
  const ta = document.getElementById('edit-json');
  if (!item) return;

  let parsed;
  try { parsed = JSON.parse(ta.value); }
  catch (e) { status.textContent = 'Invalid JSON: ' + e.message; return; }
  if (parsed === null || typeof parsed !== 'object' || Array.isArray(parsed)) {
    status.textContent = 'Top-level must be a JSON object';
    return;
  }

  status.textContent = '// saving…';
  try {
    const res = await fetch(
      `/collections/${lucyCurrentCollection.id}/items/${item._id}`,
      {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(parsed),
      }
    );
    if (!res.ok && res.status !== 204) {
      status.textContent = 'Error: ' + (await res.text());
      return;
    }
    status.textContent = '// saved';
    setTimeout(() => {
      lucyCloseEditModal();
      lucyReloadItems();
    }, 600);
  } catch (e) {
    status.textContent = 'Error: ' + e.message;
  }
}

document.addEventListener('DOMContentLoaded', () => {
  lucyToggleMode();
  lucyInitBuilder();
  lucyInitForm();
  lucyInitGenerateArt();
});
