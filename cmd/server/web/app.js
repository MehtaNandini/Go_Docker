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
  const text = el('input', { type: 'text', value: todo.title, maxlength: '200' })
  const saveBtn = el('button', { className: 'save' }, 'Save')
  const delBtn = el('button', { className: 'delete' }, 'Delete')

  saveBtn.addEventListener('click', async () => {
    const payload = { title: text.value.trim(), completed: checkbox.checked }
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
  li.appendChild(saveBtn)
  li.appendChild(delBtn)
  return li
}

document.getElementById('new-form').addEventListener('submit', async (e) => {
  e.preventDefault()
  const input = document.getElementById('title')
  const title = input.value.trim()
  if (!title) return
  const todo = await fetchJSON('/api/todos/', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ title })
  })
  const list = document.getElementById('list')
  list.appendChild(renderTodo(todo))
  input.value = ''
  input.focus()
})

loadTodos().catch(err => {
  console.error(err)
  alert(err.message)
})


