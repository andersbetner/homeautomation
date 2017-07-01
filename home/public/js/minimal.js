document.addEventListener('DOMContentLoaded', function() {
  document.getElementById('menu-href').addEventListener('click', function(e) {
    e.preventDefault();
    el = document.getElementById('menu');
    el.className = 'slide-in';
  });

  function bodyClick(e) {
    var el = document.getElementById('menu');
    var menuHref = document.getElementById('menu-href');
    if (menuHref.contains(e.target)) {
      // Clicked the "show menu link"
      return true;
    }
    if (!el.contains(e.target) && el.className.indexOf('slide-in') != -1) {
      el.className = 'slide-out';
      return true;
    }
  }

  document.getElementById('body').addEventListener('touchstart', bodyClick);
  document.getElementById('body').addEventListener('click', bodyClick);
  // make all link remain in web app mode.
  if (window.navigator.standalone) {
    var x = document.getElementsByTagName('a');
    var i;
    // make all link remain in web app mode.
    for (i = 0; i < x.length; i++) {
      var target = x[i];
      if (target.href && target.href.indexOf('http') !== -1 && target.href.indexOf(document.location.host) !==
        -1) {
        target.addEventListener('click', function(e) {
          e.preventDefault();
          if (e.target.href) {
            window.location = e.target.href;
          }
          return false;
        });
      }
    }
  }

}, false);
