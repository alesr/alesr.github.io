(function () {
  'use strict';

  const tracked = new Set();

  function hasGtag() {
    return typeof window.gtag === 'function';
  }

  function normalizePath(path) {
    if (!path) {
      return '/';
    }

    return path.endsWith('/') ? path : path + '/';
  }

  function sendEvent(name, params) {
    if (!hasGtag()) {
      return;
    }

    window.gtag('event', name, Object.assign(
      {
        page_path: window.location.pathname,
        page_title: document.title
      },
      params || {}
    ));
  }

  function sendOnce(key, name, params) {
    if (tracked.has(key)) {
      return;
    }

    tracked.add(key);
    sendEvent(name, params);
  }

  function installContactTracking() {
    const sidebar = document.querySelector('.sidebar-bottom');

    if (!sidebar) {
      return;
    }

    sidebar.addEventListener('click', function (event) {
      const link = event.target.closest('a[aria-label]');

      if (!link) {
        return;
      }

      const channel = (link.getAttribute('aria-label') || '').toLowerCase();

      sendEvent('contact_click', {
        location: 'sidebar',
        channel: channel
      });
    });
  }

  function installShareTracking() {
    document.addEventListener('click', function (event) {
      const shareLink = event.target.closest('.share-icons a');

      if (shareLink) {
        const platform = (shareLink.getAttribute('title') || shareLink.getAttribute('aria-label') || 'unknown')
          .toLowerCase();

        sendEvent('share_click', {
          platform: platform,
          location: 'post_share'
        });

        return;
      }

      const copyButton = event.target.closest('#copy-link');

      if (copyButton) {
        sendEvent('copy_link_click', {
          location: 'post_share'
        });
      }
    });
  }

  function installSearchTracking() {
    const searchInput = document.getElementById('search-input');
    const searchTrigger = document.getElementById('search-trigger');

    if (searchTrigger) {
      searchTrigger.addEventListener('click', function () {
        sendOnce('search_open', 'search_used', {
          action: 'open'
        });
      });
    }

    if (!searchInput) {
      return;
    }

    function trackSearch(action) {
      const query = (searchInput.value || '').trim();

      if (!query) {
        return;
      }

      sendOnce('search_query', 'search_used', {
        action: action,
        query_length: query.length
      });
    }

    searchInput.addEventListener('keydown', function (event) {
      if (event.key === 'Enter') {
        trackSearch('enter');
      }
    });

    searchInput.addEventListener('blur', function () {
      trackSearch('blur');
    });
  }

  function installTocTracking() {
    document.addEventListener('click', function (event) {
      const trigger = event.target.closest('.toc-trigger, #toc-solo-trigger, #toc-popup-close');

      if (trigger) {
        let action = 'toggle';

        if (trigger.id === 'toc-popup-close') {
          action = 'close';
        } else if (trigger.id === 'toc-solo-trigger') {
          action = 'open';
        }

        sendEvent('toc_interaction', {
          action: action
        });

        return;
      }

      const tocLink = event.target.closest('#toc a, #toc-popup a, .toc a');

      if (!tocLink) {
        return;
      }

      sendEvent('toc_interaction', {
        action: 'link_click'
      });
    });
  }

  function installReadDepthTracking() {
    const post = document.querySelector('article[data-toc]');

    if (!post) {
      return;
    }

    let ticking = false;
    const key = 'post_read_complete:' + window.location.pathname;

    function evaluateReadDepth() {
      const viewportBottom = window.scrollY + window.innerHeight;
      const postTop = post.offsetTop;
      const postHeight = post.offsetHeight;
      const progress = (viewportBottom - postTop) / postHeight;

      if (progress >= 0.9) {
        sendOnce(key, 'post_read_complete', {
          threshold: 90
        });
      }
    }

    window.addEventListener('scroll', function () {
      if (ticking) {
        return;
      }

      ticking = true;
      window.requestAnimationFrame(function () {
        evaluateReadDepth();
        ticking = false;
      });
    });

    evaluateReadDepth();
  }

  function installWatchlistTracking() {
    const watchlistPath = normalizePath(window.location.pathname);

    if (!watchlistPath.endsWith('/watchlist/')) {
      return;
    }

    document.addEventListener('click', function (event) {
      const link = event.target.closest('main a[href*="/posts/"]');

      if (!link) {
        return;
      }

      sendEvent('watchlist_post_click', {
        location: 'watchlist'
      });
    });
  }

  function installGiscusTracking() {
    if (!document.body) {
      return;
    }

    const observer = new MutationObserver(function () {
      const frame = document.querySelector('iframe.giscus-frame');

      if (!frame) {
        return;
      }

      sendOnce('giscus_loaded:' + window.location.pathname, 'giscus_loaded', {
        location: 'comments'
      });

      observer.disconnect();
    });

    observer.observe(document.body, {
      childList: true,
      subtree: true
    });
  }

  document.addEventListener('DOMContentLoaded', function () {
    installContactTracking();
    installShareTracking();
    installSearchTracking();
    installTocTracking();
    installReadDepthTracking();
    installWatchlistTracking();
    installGiscusTracking();
  });
})();
