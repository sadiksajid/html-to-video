import './style.css';
import {
  AddBatchJobs,
  AddJob,
  CancelJob,
  CheckReelIframes,
  DeleteJob,
  GetDiagnostics,
  GetJobs,
  GetSettings,
  OpenFile,
  OpenInExplorer,
  RetryJob,
  SelectAudioFile,
  SelectBatchFiles,
  SelectMockupImage,
  SelectOutputDir,
  SelectReelFile,
  UpdateSettings
} from '../wailsjs/go/app/App';
import { jobs, settings, app, iframe } from '../wailsjs/go/models';

// Localization
type Lang = 'en' | 'de' | 'ar';

const TRANSLATIONS: Record<Lang, Record<string, string>> = {
  en: {
    appTitle: 'Reel Studio',
    appSubtitle: 'Instagram Reel Renderer for ReelSync',
    batchImport: 'Batch Import',
    reelDropTitle: 'Reel HTML File',
    reelDropSubtitle: 'Click or drop reel.html',
    audioDropTitle: 'Voice-over Audio',
    audioDropSubtitle: 'mp3, wav, m4a, aac, ogg...',
    browseFile: 'Browse...',
    resolutionLabel: 'Resolution',
    fpsLabel: 'Frame Rate',
    qualityLabel: 'Quality Preset',
    outputFolderLabel: 'Output Folder:',
    changeFolder: 'Change...',
    renderButton: 'Render Video',
    rendering: 'Rendering...',
    videosTitle: 'Videos & Queue',
    emptyQueue: 'No videos rendered yet. Drop a reel and voice-over above to begin.',
    batchModalTitle: 'Batch Import Pairs',
    batchModalDesc: 'Matched reels and audio files by name. Review pairs before queueing:',
    selectMoreFiles: 'Select Files...',
    enqueueAll: 'Add Pairs to Queue',
    cancel: 'Cancel',
    settingsTitle: 'Settings',
    langLabel: 'Language',
    concurrencyLabel: 'Max Parallel Renders',
    browserChoiceLabel: 'Browser Automation Engine',
    keepFilesLabel: 'Keep intermediate job files (temp frames & scripts)',
    saveSettings: 'Save Changes',
    close: 'Close',
    diagnosticsTitle: 'System Diagnostics & Logs',
    copyDiagnostics: 'Copy Diagnostics',
    copied: 'Copied!',
    openFile: 'Open File',
    showInExplorer: 'Show in Explorer',
    renderAgain: 'Render Again',
    delete: 'Delete',
    retry: 'Retry',
    preview: 'Preview',
    iframeModalTitle: 'Mockup Image Replacement',
    iframeModalDesc: 'The following website(s) in your reel cannot be embedded directly in an iframe (e.g. blocked by security policies like X-Frame-Options). Upload a screenshot or image to use in the mockup instead:',
    confirmReplacements: 'Use Replacement Image',
    skip: 'Skip / Keep As Is',
    chooseImage: 'Choose Screenshot...',
    imageSelected: 'Image Attached ✓',
    checkingIframes: 'Checking live website frames...',
    zoomLabel: 'Mockup Zoom',
    positionLabel: 'Image Position',
    posTop: 'Top (Header visible)',
    posCenter: 'Center',
    posBottom: 'Bottom',
    fullCoverHint: 'Image covers 100% of the mockup screen with zero borders.'
  },
  de: {
    appTitle: 'Reel Studio',
    appSubtitle: 'Instagram Reel Renderer für ReelSync',
    batchImport: 'Stapel-Import',
    reelDropTitle: 'Reel HTML-Datei',
    reelDropSubtitle: 'Klicken oder reel.html ablegen',
    audioDropTitle: 'Voice-Over Audio',
    audioDropSubtitle: 'mp3, wav, m4a, aac, ogg...',
    browseFile: 'Durchsuchen...',
    resolutionLabel: 'Auflösung',
    fpsLabel: 'Bildrate',
    qualityLabel: 'Qualitätsstufe',
    outputFolderLabel: 'Ausgabeordner:',
    changeFolder: 'Ändern...',
    renderButton: 'Video Rendern',
    rendering: 'Rendern...',
    videosTitle: 'Videos & Warteschlange',
    emptyQueue: 'Noch keine Videos gerendert. Ziehe oben ein Reel und Audio hinein.',
    batchModalTitle: 'Stapel-Import Paare',
    batchModalDesc: 'Reels und Audiodateien nach Namen zugeordnet:',
    selectMoreFiles: 'Dateien wählen...',
    enqueueAll: 'Alle zur Warteschlange hinzufügen',
    cancel: 'Abbrechen',
    settingsTitle: 'Einstellungen',
    langLabel: 'Sprache',
    concurrencyLabel: 'Max. parallele Renderings',
    browserChoiceLabel: 'Browser-Automatisierungs-Engine',
    keepFilesLabel: 'Temporäre Zwischendateien behalten',
    saveSettings: 'Speichern',
    close: 'Schließen',
    diagnosticsTitle: 'Systemdiagnose & Protokolle',
    copyDiagnostics: 'Diagnose kopieren',
    copied: 'Kopiert!',
    openFile: 'Datei öffnen',
    showInExplorer: 'Im Explorer anzeigen',
    renderAgain: 'Erneut rendern',
    delete: 'Löschen',
    retry: 'Wiederholen',
    preview: 'Vorschau',
    iframeModalTitle: 'Mockup-Bildersatz',
    iframeModalDesc: 'Die folgende(n) Webseite(n) im Reel können nicht direkt eingebettet werden (z. B. durch X-Frame-Options blockiert). Lade bitte stattdessen einen Screenshot/Bild hoch:',
    confirmReplacements: 'Ersatzbild verwenden',
    skip: 'Überspringen',
    chooseImage: 'Screenshot wählen...',
    imageSelected: 'Bild ausgewählt ✓',
    checkingIframes: 'Prüfe Webseiten-Iframes...',
    zoomLabel: 'Mockup-Zoom',
    positionLabel: 'Bildposition',
    posTop: 'Oben (Kopfzeile sichtbar)',
    posCenter: 'Zentriert',
    posBottom: 'Unten',
    fullCoverHint: 'Das Bild wird gezoomt, um den Mockup-Bildschirm randlos auszufüllen.'
  },
  ar: {
    appTitle: 'ريل ستوديو',
    appSubtitle: 'مُنشئ ومُحوّل فيديوهات إنستغرام ريلز من ReelSync',
    batchImport: 'استيراد جماعي',
    reelDropTitle: 'ملف الريل HTML',
    reelDropSubtitle: 'انقر أو اسحب ملف reel.html هنا',
    audioDropTitle: 'ملف التعليق الصوتي',
    audioDropSubtitle: 'mp3, wav, m4a, aac, ogg...',
    browseFile: 'استعراض...',
    resolutionLabel: 'الدقة',
    fpsLabel: 'معدل الإطارات',
    qualityLabel: 'مستوى الجودة',
    outputFolderLabel: 'مجلد الإخراج:',
    changeFolder: 'تغيير...',
    renderButton: 'بدء الرندرة',
    rendering: 'جارٍ الإنتاج...',
    videosTitle: 'الفيديوهات وقائمة الانتظار',
    emptyQueue: 'لم يتم إنتاج أي فيديو بعد. اسحب ملف ريل وتعليق صوتي بالأعلى للبدء.',
    batchModalTitle: 'أزواج الاستيراد الجماعي',
    batchModalDesc: 'تمت مطابقة ملفات الريل والصوت حسب الاسم. تحقق من الأزواج:',
    selectMoreFiles: 'اختيار ملفات إضافية...',
    enqueueAll: 'إضافة الأزواج إلى قائمة الانتظار',
    cancel: 'إلغاء',
    settingsTitle: 'الإعدادات',
    langLabel: 'اللغة',
    concurrencyLabel: 'أقصى عدد رندرة متزامنة',
    browserChoiceLabel: 'محرك أتمتة المتصفح',
    keepFilesLabel: 'الاحتفاظ بالملفات المؤقتة والإطارات',
    saveSettings: 'حفظ التغييرات',
    close: 'إغلاق',
    diagnosticsTitle: 'تشخيص النظام والسجلات',
    copyDiagnostics: 'نسخ التشخيص',
    copied: 'تم النسخ!',
    openFile: 'فتح الملف',
    showInExplorer: 'إظهار في مستكشف الملفات',
    renderAgain: 'إعادة الرندرة',
    delete: 'حذف',
    retry: 'إعادة المحاولة',
    preview: 'معاينة',
    iframeModalTitle: 'استبدال شاشة النموذج (Mockup) بصورة',
    iframeModalDesc: 'الموقع/المواقع التالية في الريل تمنع التضمين داخل إطار (مثل سياسة X-Frame-Options). يرجى تحميل لقطة شاشة أو صورة لاستخدامها داخل نموذج الهاتف:',
    confirmReplacements: 'استخدام الصورة البديلة',
    skip: 'تخطي / الإبقاء كما هي',
    chooseImage: 'اختيار صورة لقطة الشاشة...',
    imageSelected: 'تم إرفاق الصورة ✓',
    checkingIframes: 'جارٍ فحص مواقع الريل...',
    zoomLabel: 'مستوى التكبير (Zoom)',
    positionLabel: 'موضع الصورة',
    posTop: 'أعلى (إظهار رأس الصفحة)',
    posCenter: 'وسط',
    posBottom: 'أسفل',
    fullCoverHint: 'تغطي الصورة شاشة النموذج بالكامل دون أي حواف فارغة باستخدام التكبير.'
  }
};

// State
let currentLang: Lang = 'en';
let currentSettings: settings.Settings = new settings.Settings();
let selectedHtmlPath: string = '';
let selectedAudioPath: string = '';
let jobsMap: Map<string, jobs.Job> = new Map();
let batchPairs: app.BatchItem[] = [];
let previewVideoPath: string = '';
let iframeReplacements: Record<string, string> = {};

const $ = <T extends HTMLElement>(sel: string): T => document.querySelector(sel) as T;

function applyLanguage(lang: Lang) {
  currentLang = lang;
  document.documentElement.lang = lang;
  document.documentElement.dir = lang === 'ar' ? 'rtl' : 'ltr';

  const dict = TRANSLATIONS[lang] || TRANSLATIONS.en;
  document.querySelectorAll('[data-i18n]').forEach(el => {
    const key = el.getAttribute('data-i18n');
    if (key && dict[key]) {
      el.textContent = dict[key];
    }
  });
}

function formatETA(seconds: number): string {
  if (seconds <= 0) return '';
  if (seconds >= 60) {
    const m = Math.floor(seconds / 60);
    const s = seconds % 60;
    return `~${m} min ${s} s`;
  }
  return `~${seconds} s`;
}

function updateCheck() {
  const btn = $('#btnRender') as HTMLButtonElement;
  btn.disabled = !(selectedHtmlPath && selectedAudioPath);
}

function renderJobs() {
  const container = $('#jobsList');
  const badge = $('#queueBadge');
  const list = Array.from(jobsMap.values()).sort((a, b) => b.created - a.created);

  badge.textContent = String(list.length);

  if (list.length === 0) {
    container.innerHTML = `
      <div class="empty-state">
        <span class="empty-icon">📂</span>
        <p>${TRANSLATIONS[currentLang].emptyQueue}</p>
      </div>`;
    return;
  }

  container.innerHTML = '';
  list.forEach(j => {
    const card = document.createElement('div');
    card.className = 'job-card';
    card.id = `job-${j.id}`;

    let statusClass = '';
    let thumbContent = '';
    if (j.status === 'done') {
      statusClass = 'status-done';
      thumbContent = '✓';
    } else if (j.status === 'error' || j.status === 'interrupted') {
      statusClass = 'status-error';
      thumbContent = '!';
    } else {
      thumbContent = `${j.pct || 0}%`;
    }

    const t = TRANSLATIONS[currentLang];
    let actionsHtml = '';
    if (j.status === 'done') {
      actionsHtml = `
        <button class="btn small primary" data-preview="${j.outPath}" data-title="${j.name}">▶ ${t.preview}</button>
        <button class="btn small secondary" data-open="${j.outPath}">${t.openFile}</button>
        <button class="btn small secondary" data-reveal="${j.outPath}">${t.showInExplorer}</button>
        <button class="btn small secondary" data-retry="${j.id}">${t.renderAgain}</button>
        <button class="btn small danger" data-del="${j.id}">${t.delete}</button>
      `;
    } else if (j.status === 'error' || j.status === 'interrupted') {
      actionsHtml = `
        <button class="btn small primary" data-retry="${j.id}">${t.retry}</button>
        <button class="btn small danger" data-del="${j.id}">${t.delete}</button>
      `;
    } else {
      actionsHtml = `
        <button class="btn small danger" data-cancel="${j.id}">${t.cancel}</button>
      `;
    }

    let statusLine = '';
    if (j.status === 'done') {
      const dur = j.duration ? `${j.duration.toFixed(1)}s` : '';
      const mb = j.fileSize ? `${(j.fileSize / (1024 * 1024)).toFixed(1)} MB` : '';
      statusLine = `<div class="job-status-line success">✓ ${t.close} · ${dur} · ${mb}</div>`;
    } else if (j.status === 'error' || j.status === 'interrupted') {
      statusLine = `<div class="job-status-line error">✗ ${j.error || j.phase}</div>`;
    } else {
      const eta = j.eta ? ` · ${formatETA(j.eta)}` : '';
      const frames = j.frames ? ` · frame ${j.frame}/${j.frames}` : '';
      statusLine = `<div class="job-status-line">${j.phase}${frames}${eta}</div>`;
    }

    card.innerHTML = `
      <div class="job-thumb ${statusClass}">${thumbContent}</div>
      <div class="job-details">
        <div class="job-title" title="${j.outPath}">${j.name}</div>
        <div class="job-chips">
          <span class="chip">${j.options.resolution}p</span>
          <span class="chip">${j.options.fps} fps</span>
          <span class="chip">${j.options.quality}</span>
          ${j.iframeReplacements && Object.keys(j.iframeReplacements).length > 0 ? `<span class="chip" style="color:#34d399">🖼️ Image Mockup</span>` : ''}
        </div>
        ${j.status !== 'done' && j.status !== 'error' && j.status !== 'interrupted' ? `
          <div class="progress-bar-container">
            <div class="progress-bar" style="width: ${j.pct || 0}%"></div>
          </div>
        ` : ''}
        ${statusLine}
      </div>
      <div class="job-actions">${actionsHtml}</div>
    `;

    container.appendChild(card);
  });
}

async function loadSettings() {
  try {
    currentSettings = await GetSettings();
    applyLanguage(currentSettings.language as Lang || 'en');
    $('#txtOutputDir').textContent = currentSettings.defaultOutputDir || '...';
    ($('#selectResolution') as HTMLSelectElement).value = currentSettings.defaultResolution || '1080';
    ($('#selectFps') as HTMLSelectElement).value = String(currentSettings.defaultFps || 30);
    ($('#selectQuality') as HTMLSelectElement).value = currentSettings.defaultQuality || 'high';
    ($('#cfgLanguage') as HTMLSelectElement).value = currentSettings.language || 'en';
    ($('#cfgConcurrency') as HTMLInputElement).value = String(currentSettings.maxParallelRenders || 1);
    $('#cfgConcurrencyVal').textContent = String(currentSettings.maxParallelRenders || 1);
    ($('#cfgBrowser') as HTMLSelectElement).value = currentSettings.browserChoice || 'msedge';
    ($('#cfgKeepIntermediate') as HTMLInputElement).checked = currentSettings.keepIntermediateFiles || false;
  } catch (err) {
    console.error('Failed to load settings:', err);
  }
}

async function loadJobs() {
  try {
    const list = await GetJobs();
    jobsMap.clear();
    list.forEach(j => jobsMap.set(j.id, j));
    renderJobs();
  } catch (err) {
    console.error('Failed to load jobs:', err);
  }
}

async function checkReelIframesAndPrompt(htmlPath: string) {
  const t = TRANSLATIONS[currentLang];
  const originalLabel = $('#htmlLabel').textContent;
  $('#htmlLabel').textContent = t.checkingIframes;

  try {
    const results = await CheckReelIframes(htmlPath);
    const blocked = results.filter(r => !r.working);

    if (blocked.length > 0) {
      renderIframeModal(blocked);
      $('#modalIframe').style.display = 'flex';
    }
  } catch (err) {
    console.warn('Iframe check failed:', err);
  } finally {
    $('#htmlLabel').textContent = originalLabel;
  }
}

function toSafeId(str: string): string {
  let hash = 0;
  for (let i = 0; i < str.length; i++) {
    hash = ((hash << 5) - hash) + str.charCodeAt(i);
    hash |= 0;
  }
  return 'ifr_' + Math.abs(hash);
}

function renderIframeModal(blockedItems: iframe.CheckResult[]) {
  const container = $('#iframeBlockList');
  container.innerHTML = '';
  const t = TRANSLATIONS[currentLang];

  const getUrlState = (url: string) => {
    const raw = iframeReplacements[url];
    let img = '';
    let zoom = 1.0;
    let pos = 'top center';
    if (raw) {
      try {
        const p = JSON.parse(raw);
        if (p && typeof p === 'object') {
          img = p.image || '';
          zoom = typeof p.zoom === 'number' ? p.zoom : 1.0;
          pos = p.position || 'top center';
        } else {
          img = raw;
        }
      } catch {
        img = raw;
      }
    }
    return { img, zoom, pos };
  };

  const updateReplacement = (url: string, img: string, zoom: number, pos: string) => {
    if (!img) return;
    iframeReplacements[url] = JSON.stringify({
      image: img,
      zoom: zoom,
      position: pos
    });
  };

  blockedItems.forEach(item => {
    const row = document.createElement('div');
    row.className = 'iframe-block-item';

    const safeUrl = item.url;
    const { img: imgPath, zoom: zoomVal, pos: posVal } = getUrlState(safeUrl);
    const safeId = toSafeId(safeUrl);

    row.innerHTML = `
      <div class="iframe-header">
        <div>
          <div class="iframe-url">${safeUrl}</div>
          <div class="iframe-reason">⚠️ ${item.reason}</div>
        </div>
      </div>
      <div class="iframe-upload-row">
        <div class="replacement-preview" id="preview-${safeId}">
          ${imgPath ? `<img src="file:///${imgPath.replace(/\\/g, '/')}" style="transform: scale(${zoomVal}); transform-origin: ${posVal}; object-position: ${posVal};">` : '<span style="font-size:24px">🖼️</span>'}
        </div>
        <div class="iframe-controls">
          <div style="display: flex; align-items: center; gap: 10px;">
            <button type="button" class="btn secondary small" data-pick-img="${safeUrl}">
              ${t.chooseImage}
            </button>
            <div id="badge-${safeId}">
              ${imgPath ? `<span class="iframe-mockup-badge">${t.imageSelected}</span>` : ''}
            </div>
          </div>

          <div class="iframe-control-field">
            <label>${t.zoomLabel}</label>
            <div class="zoom-slider-row">
              <input type="range" class="zoom-slider" data-url="${safeUrl}" data-id="${safeId}" min="1.0" max="2.5" step="0.05" value="${zoomVal}">
              <span class="zoom-val-badge" id="zoom-val-${safeId}">${Math.round(zoomVal * 100)}%</span>
            </div>
          </div>

          <div class="iframe-control-field">
            <label>${t.positionLabel}</label>
            <select class="pos-select" data-url="${safeUrl}" data-id="${safeId}">
              <option value="top center" ${posVal === 'top center' ? 'selected' : ''}>${t.posTop}</option>
              <option value="center center" ${posVal === 'center center' ? 'selected' : ''}>${t.posCenter}</option>
              <option value="bottom center" ${posVal === 'bottom center' ? 'selected' : ''}>${t.posBottom}</option>
            </select>
          </div>

          <div class="mockup-cover-hint">
            ✨ ${t.fullCoverHint}
          </div>
        </div>
      </div>
    `;

    container.appendChild(row);
  });

  // Image picker event listeners
  container.querySelectorAll('[data-pick-img]').forEach(btn => {
    btn.addEventListener('click', async () => {
      const urlKey = (btn as HTMLElement).dataset.pickImg;
      if (!urlKey) return;
      try {
        const imgPath = await SelectMockupImage();
        if (imgPath) {
          const safeId = toSafeId(urlKey);
          const state = getUrlState(urlKey);
          state.img = imgPath;
          updateReplacement(urlKey, state.img, state.zoom, state.pos);

          const prev = $(`#preview-${safeId}`);
          if (prev) {
            prev.innerHTML = `<img src="file:///${imgPath.replace(/\\/g, '/')}" style="transform: scale(${state.zoom}); transform-origin: ${state.pos}; object-position: ${state.pos};">`;
          }
          const badgeBox = $(`#badge-${safeId}`);
          if (badgeBox) {
            badgeBox.innerHTML = `<span class="iframe-mockup-badge">${t.imageSelected}</span>`;
          }
        }
      } catch (err) {}
    });
  });

  // Zoom slider event listeners
  container.querySelectorAll('.zoom-slider').forEach(slider => {
    slider.addEventListener('input', e => {
      const input = e.target as HTMLInputElement;
      const urlKey = input.dataset.url;
      const safeId = input.dataset.id;
      if (!urlKey || !safeId) return;

      const zoom = parseFloat(input.value);
      const valBadge = $(`#zoom-val-${safeId}`);
      if (valBadge) valBadge.textContent = `${Math.round(zoom * 100)}%`;

      const state = getUrlState(urlKey);
      state.zoom = zoom;
      updateReplacement(urlKey, state.img, state.zoom, state.pos);

      const previewImg = $(`#preview-${safeId} img`) as HTMLImageElement;
      if (previewImg) {
        previewImg.style.transform = `scale(${zoom})`;
      }
    });
  });

  // Position select event listeners
  container.querySelectorAll('.pos-select').forEach(sel => {
    sel.addEventListener('change', e => {
      const select = e.target as HTMLSelectElement;
      const urlKey = select.dataset.url;
      const safeId = select.dataset.id;
      if (!urlKey || !safeId) return;

      const pos = select.value;
      const state = getUrlState(urlKey);
      state.pos = pos;
      updateReplacement(urlKey, state.img, state.zoom, state.pos);

      const previewImg = $(`#preview-${safeId} img`) as HTMLImageElement;
      if (previewImg) {
        previewImg.style.objectPosition = pos;
        previewImg.style.transformOrigin = pos;
      }
    });
  });
}

function setupEvents() {
  // Wails Event Bus
  const w = window as any;
  if (w.runtime && w.runtime.EventsOn) {
    w.runtime.EventsOn('job:update', (updated: jobs.Job) => {
      jobsMap.set(updated.id, updated);
      renderJobs();
    });
  }

  // Polling fallback
  setInterval(loadJobs, 2500);

  // File pickers
  $('#btnPickHtml').addEventListener('click', async () => {
    try {
      const res = await SelectReelFile();
      if (res) {
        selectedHtmlPath = res;
        $('#htmlLabel').textContent = res.split(/[\\/]/).pop() || res;
        $('#dropHtml').classList.add('active');
        updateCheck();
        await checkReelIframesAndPrompt(res);
      }
    } catch (e) {}
  });

  $('#btnPickAudio').addEventListener('click', async () => {
    try {
      const res = await SelectAudioFile();
      if (res) {
        selectedAudioPath = res;
        $('#audioLabel').textContent = res.split(/[\\/]/).pop() || res;
        $('#dropAudio').classList.add('active');
        updateCheck();
      }
    } catch (e) {}
  });

  // Drag and Drop
  ['dragenter', 'dragover'].forEach(name => {
    document.addEventListener(name, e => e.preventDefault());
  });

  const wireDrop = (el: HTMLElement, onDrop: (file: File) => void) => {
    el.addEventListener('dragover', () => el.classList.add('dragover'));
    el.addEventListener('dragleave', () => el.classList.remove('dragover'));
    el.addEventListener('drop', e => {
      e.preventDefault();
      el.classList.remove('dragover');
      if (e.dataTransfer && e.dataTransfer.files.length > 0) {
        onDrop(e.dataTransfer.files[0]);
      }
    });
  };

  wireDrop($('#dropHtml'), async f => {
    if (f.name.match(/\.html?$/i)) {
      const fPath = (f as any).path || f.name;
      selectedHtmlPath = fPath;
      $('#htmlLabel').textContent = f.name;
      $('#dropHtml').classList.add('active');
      updateCheck();
      if ((f as any).path) {
        await checkReelIframesAndPrompt((f as any).path);
      }
    }
  });

  wireDrop($('#dropAudio'), f => {
    const fPath = (f as any).path || f.name;
    selectedAudioPath = fPath;
    $('#audioLabel').textContent = f.name;
    $('#dropAudio').classList.add('active');
    updateCheck();
  });

  // Output folder change
  $('#btnChangeOutputDir').addEventListener('click', async () => {
    const dir = await SelectOutputDir();
    if (dir) {
      currentSettings.defaultOutputDir = dir;
      $('#txtOutputDir').textContent = dir;
      await UpdateSettings(currentSettings);
    }
  });

  // Render button
  $('#btnRender').addEventListener('click', async () => {
    const errBox = $('#formError');
    errBox.style.display = 'none';
    const btn = $('#btnRender') as HTMLButtonElement;
    btn.disabled = true;
    btn.textContent = TRANSLATIONS[currentLang].rendering;

    try {
      const res = ($('#selectResolution') as HTMLSelectElement).value;
      const fps = parseInt(($('#selectFps') as HTMLSelectElement).value, 10);
      const quality = ($('#selectQuality') as HTMLSelectElement).value;

      const job = await AddJob(
        selectedHtmlPath,
        selectedAudioPath,
        res,
        fps,
        quality,
        currentSettings.defaultOutputDir,
        iframeReplacements
      );

      jobsMap.set(job.id, job);
      renderJobs();

      // Reset selection
      selectedHtmlPath = '';
      selectedAudioPath = '';
      iframeReplacements = {};
      $('#dropHtml').classList.remove('active');
      $('#dropAudio').classList.remove('active');
      $('#htmlLabel').textContent = TRANSLATIONS[currentLang].reelDropSubtitle;
      $('#audioLabel').textContent = TRANSLATIONS[currentLang].audioDropSubtitle;
    } catch (err: any) {
      errBox.textContent = err?.message || String(err);
      errBox.style.display = 'block';
    } finally {
      btn.textContent = TRANSLATIONS[currentLang].renderButton;
      updateCheck();
    }
  });

  // Job actions delegation
  $('#jobsList').addEventListener('click', async e => {
    const target = e.target as HTMLElement;
    const cancelId = target.dataset.cancel;
    const delId = target.dataset.del;
    const retryId = target.dataset.retry;
    const openPath = target.dataset.open;
    const revealPath = target.dataset.reveal;
    const previewPath = target.dataset.preview;

    if (cancelId) {
      await CancelJob(cancelId);
      await loadJobs();
    } else if (delId) {
      await DeleteJob(delId);
      jobsMap.delete(delId);
      renderJobs();
    } else if (retryId) {
      await RetryJob(retryId);
      await loadJobs();
    } else if (openPath) {
      await OpenFile(openPath);
    } else if (revealPath) {
      await OpenInExplorer(revealPath);
    } else if (previewPath) {
      previewVideoPath = previewPath;
      $('#videoModalTitle').textContent = target.dataset.title || 'Video Preview';
      const player = $('#previewPlayer') as HTMLVideoElement;
      player.src = previewPath.startsWith('http') ? previewPath : 'file:///' + previewPath.replace(/\\/g, '/');
      $('#modalVideo').style.display = 'flex';
      player.play().catch(() => {});
    }
  });

  // Modal close handlers
  document.querySelectorAll('[data-close]').forEach(btn => {
    btn.addEventListener('click', () => {
      const modalId = (btn as HTMLElement).dataset.close;
      if (modalId) {
        $(`#${modalId}`).style.display = 'none';
        if (modalId === 'modalVideo') {
          const p = $('#previewPlayer') as HTMLVideoElement;
          p.pause();
          p.src = '';
        }
      }
    });
  });

  // Iframe modal confirm
  $('#btnConfirmIframeReplacements').addEventListener('click', () => {
    $('#modalIframe').style.display = 'none';
  });

  // Video preview buttons inside modal
  $('#btnOpenVideoFile').addEventListener('click', () => {
    if (previewVideoPath) OpenFile(previewVideoPath);
  });
  $('#btnRevealVideo').addEventListener('click', () => {
    if (previewVideoPath) OpenInExplorer(previewVideoPath);
  });

  // Batch import
  $('#btnBatch').addEventListener('click', () => {
    $('#modalBatch').style.display = 'flex';
    renderBatchModal();
  });

  $('#btnSelectBatchFiles').addEventListener('click', async () => {
    const res = await SelectBatchFiles();
    if (res && res.pairs) {
      batchPairs = res.pairs;
      renderBatchModal(res.unpairs);
    }
  });

  $('#btnEnqueueBatch').addEventListener('click', async () => {
    if (batchPairs.length > 0) {
      await AddBatchJobs(batchPairs);
      $('#modalBatch').style.display = 'none';
      batchPairs = [];
      await loadJobs();
    }
  });

  // Settings
  $('#btnSettings').addEventListener('click', () => {
    $('#modalSettings').style.display = 'flex';
  });

  $('#cfgConcurrency').addEventListener('input', e => {
    $('#cfgConcurrencyVal').textContent = (e.target as HTMLInputElement).value;
  });

  $('#btnSaveSettings').addEventListener('click', async () => {
    currentSettings.language = ($('#cfgLanguage') as HTMLSelectElement).value;
    currentSettings.maxParallelRenders = parseInt(($('#cfgConcurrency') as HTMLInputElement).value, 10);
    currentSettings.browserChoice = ($('#cfgBrowser') as HTMLSelectElement).value;
    currentSettings.keepIntermediateFiles = ($('#cfgKeepIntermediate') as HTMLInputElement).checked;

    await UpdateSettings(currentSettings);
    applyLanguage(currentSettings.language as Lang);
    $('#modalSettings').style.display = 'none';
  });

  // Diagnostics
  $('#btnDiagnostics').addEventListener('click', async () => {
    const diag = await GetDiagnostics();
    $('#txtDiagnostics').textContent = diag;
    $('#modalDiagnostics').style.display = 'flex';
  });

  $('#btnCopyDiagnostics').addEventListener('click', () => {
    const text = $('#txtDiagnostics').textContent || '';
    navigator.clipboard.writeText(text);
    const btn = $('#btnCopyDiagnostics');
    const old = btn.textContent;
    btn.textContent = TRANSLATIONS[currentLang].copied;
    setTimeout(() => { btn.textContent = old; }, 1500);
  });
}

function renderBatchModal(unpaired?: string[]) {
  const container = $('#batchPairsList');
  const unpairedBox = $('#batchUnpairsList');
  const enqueueBtn = $('#btnEnqueueBatch') as HTMLButtonElement;

  container.innerHTML = '';
  unpairedBox.innerHTML = '';

  if (batchPairs.length === 0) {
    container.innerHTML = '<p class="subtext">No pairs matched yet. Click "Select Files..." to choose HTML and audio files.</p>';
    enqueueBtn.disabled = true;
  } else {
    enqueueBtn.disabled = false;
    batchPairs.forEach((p, idx) => {
      const row = document.createElement('div');
      row.className = 'batch-item-row';
      const rName = p.reelPath.split(/[\\/]/).pop();
      const aName = p.audioPath.split(/[\\/]/).pop();
      row.innerHTML = `
        <div class="batch-item-names">
          <b>📄 ${rName}</b>
          <small>🎙️ ${aName}</small>
        </div>
        <button class="btn small danger" data-remove-batch="${idx}">Remove</button>
      `;
      container.appendChild(row);
    });
  }

  if (unpaired && unpaired.length > 0) {
    unpairedBox.innerHTML = `<b>Unpaired files (${unpaired.length}):</b><br>` +
      unpaired.map(f => f.split(/[\\/]/).pop()).join(', ');
  }

  container.querySelectorAll('[data-remove-batch]').forEach(btn => {
    btn.addEventListener('click', () => {
      const idx = parseInt((btn as HTMLElement).dataset.removeBatch || '0', 10);
      batchPairs.splice(idx, 1);
      renderBatchModal(unpaired);
    });
  });
}

// Bootstrap
window.addEventListener('DOMContentLoaded', async () => {
  setupEvents();
  await loadSettings();
  await loadJobs();
});
