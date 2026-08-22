(function () {
  'use strict';

  var root = document.documentElement;
  var body = document.body;
  var menuButton = document.querySelector('[data-menu-toggle]');
  var sidebar = document.querySelector('[data-sidebar]');
  var overlay = document.querySelector('[data-nav-overlay]');
  var themeButton = document.querySelector('[data-theme-toggle]');
  var dialog = document.querySelector('[data-search-dialog]');
  var searchInput = document.querySelector('[data-search-input]');
  var results = document.querySelector('[data-search-results]');
  var status = document.querySelector('[data-search-status]');
  var searchIndex = null;

  function closeMenu() {
    sidebar.classList.remove('is-open');
    overlay.classList.remove('is-open');
    menuButton.setAttribute('aria-expanded', 'false');
  }

  menuButton.addEventListener('click', function () {
    var open = !sidebar.classList.contains('is-open');
    sidebar.classList.toggle('is-open', open);
    overlay.classList.toggle('is-open', open);
    menuButton.setAttribute('aria-expanded', String(open));
  });
  overlay.addEventListener('click', closeMenu);

  themeButton.addEventListener('click', function () {
    var next = root.dataset.theme === 'dark' ? 'light' : 'dark';
    root.dataset.theme = next;
    localStorage.setItem('cloudivision-docs-theme', next);
  });

  function openSearch() {
    if (!dialog.open) dialog.showModal();
    searchInput.focus();
    if (!searchIndex) {
      fetch(body.dataset.searchUrl)
        .then(function (response) {
          if (!response.ok) throw new Error('Search index unavailable');
          return response.json();
        })
        .then(function (items) {
          searchIndex = items;
          if (searchInput.value.trim().length >= 2) {
            searchInput.dispatchEvent(new Event('input'));
          }
        })
        .catch(function () { status.textContent = 'Search index could not be loaded.'; });
    }
  }

  function closeSearch() {
    dialog.close();
    searchInput.value = '';
    results.replaceChildren();
    status.textContent = 'Start typing to search the documentation.';
  }

  document.querySelector('[data-search-open]').addEventListener('click', openSearch);
  document.querySelector('[data-search-close]').addEventListener('click', closeSearch);
  dialog.addEventListener('click', function (event) {
    if (event.target === dialog) closeSearch();
  });

  document.addEventListener('keydown', function (event) {
    var tag = event.target.tagName;
    var typing = tag === 'INPUT' || tag === 'TEXTAREA' || event.target.isContentEditable;
    if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === 'k') {
      event.preventDefault();
      openSearch();
    } else if (event.key === '/' && !typing) {
      event.preventDefault();
      openSearch();
    } else if (event.key === 'Escape' && sidebar.classList.contains('is-open')) {
      closeMenu();
    }
  });

  function excerpt(content, query) {
    var clean = content.replace(/\s+/g, ' ').trim();
    var index = clean.toLowerCase().indexOf(query);
    var start = Math.max(0, index - 55);
    var text = clean.slice(start, start + 150);
    return (start > 0 ? '…' : '') + text + (clean.length > start + 150 ? '…' : '');
  }

  searchInput.addEventListener('input', function () {
    var query = searchInput.value.trim().toLowerCase();
    results.replaceChildren();
    if (query.length < 2) {
      status.textContent = 'Enter at least two characters.';
      return;
    }
    if (!searchIndex) {
      status.textContent = 'Loading search index…';
      return;
    }

    var matches = searchIndex.filter(function (item) {
      return item.title.toLowerCase().includes(query) || item.content.toLowerCase().includes(query);
    }).slice(0, 10);
    status.textContent = matches.length ? matches.length + ' result' + (matches.length === 1 ? '' : 's') : 'No results found.';

    matches.forEach(function (item) {
      var listItem = document.createElement('li');
      var link = document.createElement('a');
      var title = document.createElement('strong');
      var summary = document.createElement('span');
      listItem.className = 'search-result';
      link.href = item.url;
      title.textContent = item.title;
      summary.textContent = excerpt(item.content, query);
      link.append(title, summary);
      listItem.append(link);
      results.append(listItem);
    });
  });

  document.querySelectorAll('.doc-content h2[id], .doc-content h3[id], .doc-content h4[id]').forEach(function (heading) {
    var anchor = document.createElement('a');
    anchor.className = 'heading-anchor';
    anchor.href = '#' + heading.id;
    anchor.setAttribute('aria-label', 'Link to ' + heading.textContent);
    anchor.textContent = '#';
    heading.append(anchor);
  });

  document.querySelectorAll('.doc-content pre').forEach(function (block) {
    var code = block.querySelector('code');
    var button = document.createElement('button');
    button.type = 'button';
    button.className = 'copy-button';
    button.textContent = 'Copy';
    button.addEventListener('click', function () {
      navigator.clipboard.writeText(code ? code.textContent : block.textContent).then(function () {
        button.textContent = 'Copied';
        window.setTimeout(function () { button.textContent = 'Copy'; }, 1400);
      });
    });
    block.append(button);
  });

  var progress = document.querySelector('.reading-progress span');
  function updateProgress() {
    var available = document.documentElement.scrollHeight - window.innerHeight;
    var percentage = available > 0 ? Math.min(100, window.scrollY / available * 100) : 0;
    progress.style.width = percentage + '%';
  }
  window.addEventListener('scroll', updateProgress, { passive: true });
  updateProgress();
}());
