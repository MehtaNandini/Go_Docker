async function fetchJSON(url, options) {
  const res = await fetch(url, options)
  if (!res.ok) {
    const err = await safeJSON(res)
    throw new Error(err && err.error ? err.error : 'Request failed')
  }
  return res.json()
}

async function safeJSON(res) {
  try {
    return await res.json()
  } catch (_) {
    return null
  }
}

function el(tag, attrs = {}, ...children) {
  const e = document.createElement(tag)
  for (const [k, v] of Object.entries(attrs)) {
    if (k === 'className') e.className = v
    else if (k === 'text') e.textContent = v
    else e.setAttribute(k, v)
  }
  for (const c of children) {
    if (typeof c === 'string') e.appendChild(document.createTextNode(c))
    else if (c) e.appendChild(c)
  }
  return e
}

async function loadTodos() {
  const list = document.getElementById('list')
  list.innerHTML = ''
  const todos = await fetchJSON('/api/todos/')
  for (const t of todos) {
    list.appendChild(renderTodo(t))
  }
}

function renderTodo(todo) {
  const li = el('li', { className: 'item' })
  const checkbox = el('input', { type: 'checkbox' })
  checkbox.checked = !!todo.completed
  const text = el('input', { type: 'text', value: todo.title, maxlength: '200', className: 'main-input' })
  const tagsInput = el('input', { type: 'text', value: formatTags(todo.tags), placeholder: 'tags', className: 'tags-input' })
  const durationInput = el('input', {
    type: 'number',
    min: '0',
    max: '1440',
    value: String(todo.durationMinutes ?? 0),
    className: 'duration-input'
  })
  const saveBtn = el('button', { className: 'save' }, 'Save')
  const delBtn = el('button', { className: 'delete' }, 'Delete')
  const priority = el('span', { className: 'priority-pill' }, `Priority ${formatPriority(todo.priorityScore)}`)

  saveBtn.addEventListener('click', async () => {
    const payload = {
      title: text.value.trim(),
      completed: checkbox.checked,
      tags: parseTags(tagsInput.value),
      durationMinutes: parseDuration(durationInput.value)
    }
    const updated = await fetchJSON(`/api/todos/${todo.id}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload)
    })
    li.replaceWith(renderTodo(updated))
  })

  delBtn.addEventListener('click', async () => {
    const ok = confirm('Delete this item?')
    if (!ok) return
    const res = await fetch(`/api/todos/${todo.id}`, { method: 'DELETE' })
    if (!res.ok) {
      const err = await safeJSON(res)
      alert(err && err.error ? err.error : 'Delete failed')
      return
    }
    li.remove()
  })

  li.appendChild(checkbox)
  li.appendChild(text)
  li.appendChild(tagsInput)
  li.appendChild(durationInput)
  li.appendChild(priority)
  li.appendChild(saveBtn)
  li.appendChild(delBtn)
  return li
}

document.getElementById('new-form').addEventListener('submit', async (e) => {
  e.preventDefault()
  const input = document.getElementById('title')
  const tagsInput = document.getElementById('tags')
  const durationInput = document.getElementById('duration')
  const title = input.value.trim()
  if (!title) return
  const todo = await fetchJSON('/api/todos/', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      title,
      tags: parseTags(tagsInput.value),
      durationMinutes: parseDuration(durationInput.value)
    })
  })
  const list = document.getElementById('list')
  list.appendChild(renderTodo(todo))
  input.value = ''
  tagsInput.value = ''
  durationInput.value = ''
  input.focus()
})

function parseTags(value) {
  if (!value) return []
  return value
    .split(',')
    .map(t => t.trim().toLowerCase())
    .filter(Boolean)
    .slice(0, 10)
}

function formatTags(tags) {
  if (!Array.isArray(tags) || tags.length === 0) return ''
  return tags.join(', ')
}

function parseDuration(value) {
  const num = Number(value)
  if (Number.isNaN(num) || num < 0) return 0
  if (num > 1440) return 1440
  return Math.round(num)
}

function formatPriority(score) {
  const val = typeof score === 'number' ? score : 0
  return val.toFixed(2)
}

loadTodos().catch(err => {
  console.error(err)
  alert(err.message)
})


