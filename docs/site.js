(function () {
  'use strict';
  var $ = function (s, r) { return (r || document).querySelector(s); };
  var $$ = function (s, r) { return Array.prototype.slice.call((r || document).querySelectorAll(s)); };
  var root = document.documentElement;

  var FA = {
    'skip': 'رفتن به دمو',
    'nav.demo': 'دمو', 'nav.install': 'نصب', 'nav.how': 'چطور', 'nav.compare': 'مقایسه',
    'hero.kicker': 'ابزار خط فرمان متن‌باز · GitHub Action · اجرا در مرورگر',
    'hero.title': 'آیا اپ شما <em>از مرز رد می‌شود؟</em>',
    'hero.lede': 'marz کد شما را می‌خواند و می‌گوید کدام سرویس‌ها در ایران، چین و روسیه فیلتر، تحریم یا در دسترس نیستند: Firebase، Google Maps، reCAPTCHA، OpenAI، Stripe، یوتیوب و بقیه. برای هر حکم منبع می‌آورد و جایگزینی پیشنهاد می‌دهد که آنجا کار می‌کند.',
    'copy': 'کپی', 'copied': 'کپی شد ✓',
    'install.fine': 'یک فایل اجرایی تکی بدون وابستگی. نصب‌کننده دانلودِ قطع‌شده را ادامه می‌دهد و SHA-256 را بررسی می‌کند. بعد در پوشهٔ پروژه بزنید: marz',
    'board.title': 'پروازها · اپ شما', 'board.service': 'سرویس', 'board.to': 'مقصد', 'board.status': 'وضعیت',
    'board.foot': 'وضعیت‌ها از پایگاه دانش marz، به‌روزشده در ۲۵ مهر ۱۴۰۵',
    'demo.title': 'همین‌جا اسکن کنید',
    'demo.sub': 'این همان اسکنر واقعی marz است که به WebAssembly کامپایل شده. فایل‌های نمونه را ویرایش کنید، کد خودتان را بچسبانید یا یک پوشه باز کنید. چیزی آپلود نمی‌شود و همه‌چیز در مرورگر خودتان اجرا می‌شود.',
    'demo.folder': 'باز کردن پوشه…', 'demo.reset': 'بازگشت به نمونه', 'demo.loading': 'در حال بارگذاری اسکنر…',
    'demo.ready': 'اسکنر آماده است · WebAssembly', 'demo.copymd': 'کپی گزارش Markdown', 'demo.fail': 'بارگذاری اسکنر ممکن نشد. صفحه را تازه کنید.',
    'demo.files': '{n} فایل اسکن شد', 'demo.none': 'هیچ سرویس حساسی از پایگاه دانش پیدا نشد. ✓',
    'demo.where': 'محل', 'demo.alts': 'جایگزین', 'demo.fix': 'راه‌حل', 'demo.src': 'منبع',
    'cc.IR': 'ایران', 'cc.CN': 'چین', 'cc.RU': 'روسیه',
    'stamp.bad': 'رد', 'stamp.mid': 'با مشکل', 'stamp.good': 'مجاز',
    'how.title': 'چهار فرمان',
    'how.scan': 'کد و ۱۲ نوع فایل مانیفست را اسکن می‌کند: npm، pip، Poetry، Go، Flutter، Gradle، Maven، CocoaPods، Composer، Bundler، Dockerfile و compose. خروجی متن، JSON، Markdown یا SARIF است و با --fail-on در CI به کار می‌آید.',
    'how.probe': 'از داخل کشور اجرایش کنید. DNS، TCP، TLS و HTTP را مرحله‌به‌مرحله آزمایش می‌کند و پاسخ DNS فیلترینگ (10.10.34.x)، صفحهٔ «پیوندها»، خطای 1009 کلادفلر، بلاک CloudFront و خطای «کشور پشتیبانی نمی‌شود» APIهای هوش مصنوعی را تشخیص می‌دهد.',
    'how.check': 'شواهد زنده از اندازه‌گیری‌های OONI در داخل هر کشور. خروجی واقعی ۲۵ مهر ۱۴۰۵:',
    'how.fix': 'رایج‌ترین مشکل را خودش حل می‌کند: فونت‌های گوگل را در پوشهٔ وب شما ذخیره و لینک‌ها و @importها را بازنویسی می‌کند.',
    'legend.title': 'معنی وضعیت‌ها',
    'legend.blocked': 'توسط فیلترینگ حکومتی مسدود شده.', 'legend.sanctioned': 'خود سرویس کاربران یا توسعه‌دهندگان آن کشور را رد می‌کند.',
    'legend.unavailable': 'سرویس در آن کشور فعال نیست یا فروخته نمی‌شود.', 'legend.degraded': 'کند، محدودشده یا ناپایدار.',
    'legend.unknown': 'دادهٔ عمومی معتبری نیست. marz به‌جای حدس زدن همین را می‌گوید؛ با probe آزمایش کنید.', 'legend.ok': 'طبق منابع عمومی کار می‌کند.',
    'action.title': 'در هر Pull Request',
    'action.sub': 'اکشن گیت‌هاب گزارش Markdown را به خلاصهٔ اجرا اضافه می‌کند، می‌تواند SARIF را در Code Scanning آپلود کند و در شدتی که خودتان انتخاب کنید بیلد را متوقف می‌کند.',
    'cmp.title': 'ابزار مشابهی هست؟',
    'cmp.sub': 'نزدیک‌ترین ابزار فقط سایت منتشرشده را آن هم فقط برای چین بررسی می‌کند. ابزار دیگری پیدا نکردیم که مخزن کد را بخواند و تحریم را هم کنار فیلترینگ پوشش دهد.',
    'cmp.code': 'کد را می‌خواند', 'cmp.cc': 'کشورها', 'cmp.sanc': 'تحریم هم', 'cmp.alt': 'منبع و جایگزین', 'cmp.probe': 'آزمایش زنده', 'cmp.oss': 'متن‌باز',
    'cmp.marzcode': 'کد + ۱۲ مانیفست', 'cmp.css': 'HTML/JS سایت منتشرشده', 'cmp.list': '✗ دامنه‌ها را خودتان می‌دهید',
    'cmp.limits': 'محدودیت‌ها: marz حدود ۳۰ سرویس را می‌شناسد، نه همه‌چیز را. فیلترینگ تغییر می‌کند، پس «نامعلوم» یعنی با probe آزمایش کنید. این ابزار مشاورهٔ حقوقی دربارهٔ تحریم نیست.'
  };
  var EN = {};
  var ST_FA = { blocked: 'فیلتر', sanctioned: 'تحریم', unavailable: 'ناموجود', degraded: 'ناپایدار', unknown: 'نامعلوم', ok: 'سالم' };
  var SEV_FA = { critical: 'بحرانی', high: 'زیاد', medium: 'متوسط', low: 'کم', info: 'اطلاع' };

  $$('[data-i18n]').forEach(function (el) { EN[el.dataset.i18n] = el.textContent; });
  $$('[data-i18n-html]').forEach(function (el) { EN[el.dataset.i18nHtml] = el.innerHTML; });
  EN['copied'] = 'Copied ✓'; EN['demo.ready'] = 'Scanner ready · WebAssembly'; EN['demo.fail'] = 'Could not load the scanner. Refresh the page.';
  EN['demo.files'] = '{n} files scanned'; EN['demo.none'] = 'No region-sensitive services from the knowledge base found. ✓';
  EN['demo.where'] = 'Where'; EN['demo.alts'] = 'Alternatives'; EN['demo.fix'] = 'Fix'; EN['demo.src'] = 'source';
  EN['stamp.bad'] = 'DENIED'; EN['stamp.mid'] = 'ISSUES'; EN['stamp.good'] = 'CLEARED';

  function lang() { return root.lang === 'fa' ? 'fa' : 'en'; }
  function t(k) { var d = lang() === 'fa' ? FA : EN; return d[k] != null ? d[k] : (EN[k] || k); }
  function applyLang() {
    $$('[data-i18n]').forEach(function (el) { el.textContent = t(el.dataset.i18n); });
    $$('[data-i18n-html]').forEach(function (el) { el.innerHTML = t(el.dataset.i18nHtml); });
    document.title = lang() === 'fa' ? 'مرز (marz) · آیا اپ شما در ایران، چین و روسیه کار می‌کند؟' : 'marz · will your app work in Iran, China and Russia?';
    if (lastResult) render(lastResult);
    if (window.marz) $('#wasm-status').textContent = t('demo.ready');
    buildBoard(true);
  }
  $('#lang').addEventListener('click', function () {
    var fa = lang() !== 'fa';
    root.lang = fa ? 'fa' : 'en'; root.dir = fa ? 'rtl' : 'ltr';
    localStorage.setItem('marz-lang', root.lang);
    applyLang();
  });
  $('#theme').addEventListener('click', function () {
    var n = root.dataset.theme === 'dark' ? 'light' : 'dark';
    root.dataset.theme = n; localStorage.setItem('marz-theme', n);
  });

  // install tabs + copy
  $$('.tabs button').forEach(function (b) {
    b.addEventListener('click', function () {
      $$('.tabs button').forEach(function (x) { x.setAttribute('aria-selected', x === b); });
      $('#cmd').textContent = b.dataset.cmd;
    });
  });
  if (/Win/.test(navigator.platform || navigator.userAgent)) $$('.tabs button')[1].click();
  function copyText(text, btn) {
    var done = function () { var o = btn.textContent; btn.textContent = t('copied'); setTimeout(function () { btn.textContent = o === t('copied') ? t('copy') : o; }, 1400); };
    if (navigator.clipboard) navigator.clipboard.writeText(text).then(done, done);
    else { var ta = document.createElement('textarea'); ta.value = text; document.body.appendChild(ta); ta.select(); try { document.execCommand('copy'); } catch (e) {} ta.remove(); done(); }
  }
  $$('[data-copy]').forEach(function (b) { b.addEventListener('click', function () { copyText($(b.dataset.copy).textContent, b); }); });

  // departures board
  var BOARD = [
    ['FIREBASE', 'IR', 'sanctioned'], ['GOOGLE FONTS', 'CN', 'degraded'], ['OPENAI API', 'RU', 'unavailable'],
    ['GOOGLE MAPS', 'IR', 'sanctioned'], ['RECAPTCHA', 'CN', 'blocked'], ['YOUTUBE', 'IR', 'blocked'],
    ['STRIPE', 'RU', 'unavailable'], ['TELEGRAM', 'RU', 'ok'], ['JSDELIVR', 'IR', 'ok'], ['GEMINI API', 'CN', 'unavailable']
  ];
  var COL = { blocked: 'var(--red)', sanctioned: 'var(--mag)', unavailable: 'var(--orange)', degraded: 'var(--yellow)', unknown: 'var(--grey)', ok: 'var(--green)' };
  var boardStart = 0, NAME_W = 12, ST_W = 11;
  var reduce = matchMedia('(prefers-reduced-motion: reduce)').matches;
  function flaps(el, text, width, color) {
    text = (text + ' '.repeat(width)).slice(0, width);
    if (el.children.length !== width) { el.innerHTML = ''; for (var i = 0; i < width; i++) { var s = document.createElement('span'); s.className = 'flap'; el.appendChild(s); } }
    Array.prototype.forEach.call(el.children, function (s, i) {
      var ch = text[i] === ' ' ? '\u00a0' : text[i];
      if (color) s.style.setProperty('--c', color);
      if (s.textContent === ch) return;
      if (reduce) { s.textContent = ch; return; }
      var steps = 2 + (i % 4), letters = 'ABCDEFGHIJKLMNOPQRSTUVWXYZ', n = 0;
      (function tick() {
        s.classList.remove('go'); void s.offsetWidth; s.classList.add('go');
        s.textContent = n++ < steps ? letters[(Math.random() * 26) | 0] : ch;
        if (n <= steps) setTimeout(tick, 70);
      })();
    });
  }
  function buildBoard(instant) {
    var rows = $('#board-rows'), small = innerWidth < 420;
    NAME_W = small ? 10 : 12; ST_W = small ? 9 : 11;
    if (!rows.children.length) for (var i = 0; i < 5; i++) rows.insertAdjacentHTML('beforeend', '<div class="row"><div class="flaps name"></div><div class="to"></div><div class="flaps stat"></div></div>');
    Array.prototype.forEach.call(rows.children, function (r, i) {
      var d = BOARD[(boardStart + i) % BOARD.length];
      var st = lang() === 'fa' ? d[2] : d[2];
      flaps($('.name', r), d[0], NAME_W);
      $('.to', r).textContent = d[1];
      flaps($('.stat', r), st.toUpperCase(), ST_W, COL[d[2]]);
    });
  }
  buildBoard();
  setInterval(function () { if (document.hidden) return; boardStart = (boardStart + 1) % BOARD.length; buildBoard(); }, 4200);
  addEventListener('resize', function () { var rows = $('#board-rows'); var small = innerWidth < 420; if ((small ? 10 : 12) !== NAME_W) { rows.innerHTML = ''; buildBoard(); } });
  (function clock() { var d = new Date(); $('#clock').textContent = ('0' + d.getHours()).slice(-2) + ':' + ('0' + d.getMinutes()).slice(-2); setTimeout(clock, 15000); })();

  // demo
  var SAMPLE = [
    { path: 'public/index.html', content: '<!doctype html>\n<html lang="en">\n<head>\n  <link rel="preconnect" href="https://fonts.googleapis.com">\n  <link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;700&display=swap" rel="stylesheet">\n  <script async src="https://www.googletagmanager.com/gtag/js?id=G-XXXX"></script>\n  <script src="https://www.google.com/recaptcha/api.js" async defer></script>\n  <script src="https://cdn.jsdelivr.net/npm/alpinejs@3/dist/cdn.min.js" defer></script>\n</head>\n<body>\n  <iframe src="https://www.youtube-nocookie.com/embed/xyz"></iframe>\n  <form><div class="g-recaptcha" data-sitekey="key"></div></form>\n</body>\n</html>\n' },
    { path: 'package.json', content: '{\n  "name": "shop",\n  "dependencies": {\n    "firebase": "^11.0.0",\n    "@react-google-maps/api": "^2.20.0",\n    "@stripe/stripe-js": "^5.0.0",\n    "@sentry/react": "^8.0.0",\n    "react": "^19.0.0"\n  }\n}\n' },
    { path: 'server/requirements.txt', content: 'fastapi==0.115.0\nopenai>=1.50\nstripe\n' },
    { path: 'app/pubspec.yaml', content: 'name: shop_app\ndependencies:\n  flutter:\n    sdk: flutter\n  google_maps_flutter: ^2.9.0\n  firebase_messaging: ^15.0.0\n' },
    { path: 'Dockerfile', content: 'FROM node:22-alpine AS build\nRUN npm ci && npm run build\n\nFROM python:3.12-slim\nCOPY --from=build /app/dist /srv/static\n' }
  ];
  var files = SAMPLE.map(function (f) { return { path: f.path, content: f.content }; });
  var cur = 0, lastResult = null, lastMd = '';
  var ta = $('#src');
  function tabs() {
    var box = $('#file-tabs'); box.innerHTML = '';
    files.slice(0, 60).forEach(function (f, i) {
      var b = document.createElement('button'); b.type = 'button'; b.setAttribute('role', 'tab');
      b.setAttribute('aria-selected', i === cur); b.textContent = f.path; b.title = f.path;
      b.onclick = function () { cur = i; tabs(); ta.value = files[cur].content; };
      box.appendChild(b);
    });
    if (files.length > 60) box.insertAdjacentHTML('beforeend', '<button type="button" disabled>+' + (files.length - 60) + '</button>');
  }
  tabs(); ta.value = files[0].content;
  var timer;
  ta.addEventListener('input', function () { files[cur].content = ta.value; clearTimeout(timer); timer = setTimeout(scan, 250); });
  $('#reset').onclick = function () { files = SAMPLE.map(function (f) { return { path: f.path, content: f.content }; }); cur = 0; tabs(); ta.value = files[0].content; scan(); };
  $$('.chip').forEach(function (c) { c.addEventListener('click', function () { var on = $$('.chip.on'); if (c.classList.contains('on') && on.length === 1) return; c.classList.toggle('on'); scan(); }); });

  var SKIP = /(^|\/)(node_modules|\.git|vendor|\.venv|venv|dist|build|\.next|Pods|\.gradle|\.dart_tool|target|__pycache__)\//;
  var BIN = /\.(png|jpe?g|gif|webp|ico|svg|woff2?|ttf|otf|eot|mp4|webm|mp3|pdf|zip|gz|tgz|jar|so|dll|exe|wasm|map|lock)$/i;
  $('#folder').addEventListener('change', function (e) {
    var list = Array.prototype.filter.call(e.target.files, function (f) { var p = f.webkitRelativePath || f.name; return !SKIP.test(p) && !BIN.test(p) && f.size < 1 << 20; }).slice(0, 4000);
    if (!list.length) return;
    $('#wasm-status').textContent = '…';
    Promise.all(list.map(function (f) { return f.text().then(function (c) { var p = (f.webkitRelativePath || f.name).split('/').slice(1).join('/') || f.name; return { path: p, content: c }; }, function () { return null; }); }))
      .then(function (r) { files = r.filter(Boolean); cur = 0; tabs(); ta.value = files.length ? files[0].content : ''; scan(); $('#wasm-status').textContent = t('demo.ready'); });
  });

  function esc(s) { return String(s).replace(/[&<>"]/g, function (c) { return { '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;' }[c]; }); }
  function scan() {
    if (!window.marz) return;
    var cc = $$('.chip.on').map(function (c) { return c.dataset.cc; }).join(',');
    var out = JSON.parse(window.marz.scan(files, cc));
    lastResult = out.result; lastMd = out.markdown;
    render(lastResult);
  }
  function stLabel(s) { return lang() === 'fa' ? ST_FA[s] || s : s; }
  function render(r) {
    var fa = lang() === 'fa';
    $('#passes').innerHTML = r.summary.map(function (s) {
      var cls = s.score >= 80 ? 'good' : s.score >= 50 ? 'mid' : 'bad';
      return '<div class="pass ' + cls + '"><div class="cc">' + s.country + ' · ' + esc(t('cc.' + s.country)) + '</div><div class="score">' + s.score + '<small>/100</small></div><span class="stamp">' + esc(t('stamp.' + cls)) + '</span></div>';
    }).join('');
    $('#files-count').textContent = t('demo.files').replace('{n}', files.length);
    if (!r.findings || !r.findings.length) { $('#findings').innerHTML = '<p class="empty">' + esc(t('demo.none')) + '</p>'; return; }
    $('#findings').innerHTML = r.findings.map(function (f) {
      var sev = f.worst_severity || 'ok';
      var v = f.verdicts.map(function (v) { return '<span class="st st-' + v.status + '">' + v.country + ' ' + esc(stLabel(v.status)) + '</span>'; }).join('');
      var locs = f.hits.slice(0, 3).map(function (h) { return esc(h.file + ':' + h.line) + ' <span>' + esc(h.match) + '</span>'; }).join('<br>') + (f.hits.length > 3 ? '<br>+' + (f.hits.length - 3) : '');
      var alts = f.verdicts.filter(function (v) { return v.severity && v.severity !== 'info'; }).map(function (v) {
        var src = (v.sources || []).slice(0, 2).map(function (s) { return '<a href="' + esc(s.url) + '" rel="noopener" target="_blank">' + esc(t('demo.src')) + '</a>'; }).join(' · ');
        return '<li><b>' + v.country + '</b> ' + esc((v.alternatives || []).join(' · ') || v.note || '') + (src ? ' <span dir="ltr">' + src + '</span>' : '') + '</li>';
      }).join('');
      return '<div class="find"><div class="find-head"><span class="sev ' + sev + '">' + esc(fa ? SEV_FA[sev] || sev : sev) + '</span><h4>' + esc(f.name) + '</h4><span class="cat">' + esc(f.category) + '</span></div>' +
        '<div class="verdicts">' + v + '</div><p>' + esc(fa ? f.why_fa : f.why) + '</p><div class="locs">' + locs + '</div>' + (alts ? '<ul class="alts">' + alts + '</ul>' : '') + '</div>';
    }).join('');
  }
  $('#copy-md').onclick = function () { if (lastMd) copyText(lastMd, $('#copy-md')); };

  window.onMarzReady = function () { $('#wasm-status').textContent = t('demo.ready'); scan(); };
  if (window.Go && window.WebAssembly) {
    var go = new Go();
    var load = WebAssembly.instantiateStreaming ? WebAssembly.instantiateStreaming(fetch('marz.wasm'), go.importObject).catch(function () { return fetch('marz.wasm').then(function (r) { return r.arrayBuffer(); }).then(function (b) { return WebAssembly.instantiate(b, go.importObject); }); })
      : fetch('marz.wasm').then(function (r) { return r.arrayBuffer(); }).then(function (b) { return WebAssembly.instantiate(b, go.importObject); });
    load.then(function (r) { go.run(r.instance); }).catch(function () { $('#wasm-status').textContent = t('demo.fail'); });
  } else { $('#wasm-status').textContent = t('demo.fail'); }

  if (lang() === 'fa') applyLang();
})();
