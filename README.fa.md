<p align="center"><img src="docs/banner.svg" alt="marz (مرز): will your app work in Iran, China and Russia?" width="100%"></p>

<div dir="rtl">

# مرز (marz)

**آیا اپ شما در ایران، چین و روسیه کار می‌کند؟**

marz کد پروژه را می‌گردد و سرویس‌هایی را پیدا می‌کند که در این کشورها **فیلتر**، **تحریم** یا **در دسترس نیستند**، مثل Firebase، Google Maps، reCAPTCHA، OpenAI، Stripe و ویدیوهای یوتیوب. برای هر کدام می‌گوید چرا، منبع عمومی می‌آورد، جایگزین ایرانی و جهانی پیشنهاد می‌دهد و می‌تواند دسترسی را از داخل کشور آزمایش کند.

[وب‌سایت و دموی زنده](https://mrzroot.github.io/marz/) · [English](README.md)

## چرا؟

بیشتر اپ‌ها طوری ساخته می‌شوند که انگار مرزی وجود ندارد: فرم ثبت‌نام به کپچای گوگل وابسته است، بک‌اند Firebase است، نقشه Google Maps است، پرداخت با Stripe انجام می‌شود و قابلیت هوش مصنوعی مستقیم `api.openai.com` را صدا می‌زند. اپ که در ایران منتشر شود، بخشی از آن بی‌سروصدا از کار می‌افتد و تا کاربرها شکایت نکنند کسی متوجه نمی‌شود.

marz این مشکل را قبل از انتشار پیدا می‌کند:

- **کل پروژه را آفلاین اسکن می‌کند**: آدرس‌ها و دامنه‌ها در هر فایل متنی، به‌علاوهٔ وابستگی‌های `package.json`، `requirements.txt`، `pyproject.toml`، `go.mod`، `pubspec.yaml`، Gradle، `pom.xml`، `Podfile`، `composer.json`، `Gemfile` و Dockerfile.
- **برای هر کشور حکم جداگانه با منبع می‌دهد**: `blocked` (فیلتر حکومتی)، `sanctioned` (خود سرویس کشور را رد می‌کند)، `unavailable` (در آن کشور ارائه نمی‌شود)، `degraded` (کند یا ناپایدار)، `unknown` و `ok`. هیچ حکمی بدون منبع نیست، و جایی که دادهٔ معتبری منتشر نشده marz به‌جای حدس زدن می‌گوید «نامعلوم».
- **جایگزین پیشنهاد می‌دهد**: نشان و Map.ir برای نقشه؛ زرین‌پال، زیبال و IDPay برای پرداخت؛ لیارا، آروان و همروش برای میزبانی؛ ALTCHA برای کپچا؛ GlitchTip برای ثبت خطا؛ و درگاه‌های هوش مصنوعی سازگار با OpenAI.
- **شواهد زنده نشان می‌دهد**: `marz check` و `scan --ooni` اندازه‌گیری‌های [OONI](https://ooni.org) از داخل کشور را می‌آورند. `marz probe` دسترسی را مرحله‌به‌مرحله (DNS، TCP، TLS، HTTP) آزمایش می‌کند و پاسخ DNS فیلترینگ ایران (10.10.34.x)، صفحهٔ «پیوندها»، خطای 1009 کلادفلر، بلاک کشوری CloudFront، `unsupported_country` در OpenAI و پیام «user location is not supported» گوگل را تشخیص می‌دهد.
- **رایج‌ترین مشکل را خودش درست می‌کند**: `marz fix fonts --write` فونت‌های گوگل را دانلود می‌کند، روی سایت خودتان میزبانی می‌کند و لینک‌ها را بازنویسی می‌کند.
- **برای CI آماده است**: خروجی متن، JSON، Markdown و SARIF، گزینهٔ `--fail-on` و یک GitHub Action.

## نصب

<div dir="ltr">

```sh
curl -fsSL https://mrzroot.github.io/marz/install.sh | sh
```

```powershell
irm https://mrzroot.github.io/marz/install.ps1 | iex
```

</div>

هر دو نصب‌کننده دانلودِ قطع‌شده را ادامه می‌دهند و SHA-256 را بررسی می‌کنند. marz یک فایل اجرایی تکی است که پایگاه دانشش داخل خودش است.

## استفاده

<div dir="ltr">

```sh
cd your-app
marz                         # IR, CN, RU
marz --country IR -v         # فقط ایران، با همهٔ منابع
marz --fail-on high          # در CI شکست بخورد
marz probe --from .          # آزمایش همهٔ سرویس‌های پیداشده از همین شبکه
marz check www.youtube.com --country IR
marz fix fonts --write
```

```text
$ marz check www.youtube.com --country IR,RU
IR www.youtube.com   blocked: 4762 measurements, 93% anomalous, 3250 confirmed blocks
RU www.youtube.com   likely blocked: 49598 measurements, 55% anomalous, 0 confirmed blocks
```

</div>

## محدودیت‌ها

- marz فقط سرویس‌هایی را می‌شناسد که در پایگاه دانشش هستند (۳۱ سرویس در مهر ۱۴۰۵). برای بقیه از `marz probe` استفاده کنید.
- فیلترینگ مدام تغییر می‌کند. «نامعلوم» یعنی دادهٔ عمومی معتبری وجود ندارد، پس از داخل کشور آزمایش کنید.
- `probe` فقط وضعیت همان شبکه‌ای را نشان می‌دهد که روی آن اجرا می‌شود. اگر VPN یا پروکسی روشن باشد، نتیجه مربوط به همان است و marz هشدار می‌دهد.
- این ابزار مشاورهٔ حقوقی دربارهٔ تحریم نیست.

## مجوز

MIT © محمدرضا زارع (M-R-Z)

</div>
