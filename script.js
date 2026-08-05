const header = document.querySelector('[data-header]');
const nav = document.querySelector('[data-nav]');
const navToggle = document.querySelector('[data-nav-toggle]');
const navLinks = Array.from(document.querySelectorAll('.site-nav a'));
const revealItems = Array.from(document.querySelectorAll('.reveal'));
const themedSections = Array.from(document.querySelectorAll('.signal-band, .section'));
const reducedMotion = window.matchMedia('(prefers-reduced-motion: reduce)').matches;

revealItems.forEach((item, index) => {
  item.style.setProperty('--reveal-delay', `${(index % 6) * 45}ms`);
});

function updateHeaderState() {
  header?.classList.toggle('is-scrolled', window.scrollY > 24);
}

function updateSectionDepth() {
  const viewportHeight = window.innerHeight || document.documentElement.clientHeight || 1;
  const readingLine = viewportHeight * 0.54;

  themedSections.forEach(section => {
    const rect = section.getBoundingClientRect();
    const center = rect.top + rect.height / 2;
    const distance = Math.abs(center - readingLine);
    const depth = Math.max(0, Math.min(1, 1 - distance / (viewportHeight * 0.86)));

    section.style.setProperty('--section-depth', depth.toFixed(3));
    section.style.setProperty('--section-light-pct', `${Math.round(depth * 18)}%`);
    section.style.setProperty('--section-deep-pct', `${Math.round(depth * 30)}%`);
  });
}

function closeNav() {
  if (!nav || !navToggle) return;
  nav.classList.remove('is-open');
  navToggle.setAttribute('aria-expanded', 'false');
}

function updateActiveNav() {
  const scrollPosition = window.scrollY + 160;
  let activeId = '';
  const hashId = window.location.hash.slice(1);
  const hashSection = hashId ? document.getElementById(hashId) : null;
  const hashTop = hashSection?.getBoundingClientRect().top ?? Infinity;

  if (hashSection && hashTop <= 160 && hashTop + hashSection.offsetHeight > 0) {
    setActiveNav(hashId);
    return;
  }

  for (const link of navLinks) {
    const id = link.getAttribute('href')?.slice(1);
    const section = id ? document.getElementById(id) : null;
    const sectionTop = section ? section.getBoundingClientRect().top + window.scrollY : Infinity;
    if (section && sectionTop <= scrollPosition) activeId = id;
  }

  navLinks.forEach(link => {
    link.classList.toggle('is-active', link.getAttribute('href') === `#${activeId}`);
  });
}

function setActiveNav(id) {
  navLinks.forEach(link => {
    link.classList.toggle('is-active', link.getAttribute('href') === `#${id}`);
  });
}

function syncHashNav() {
  const id = window.location.hash.slice(1);
  if (id && document.getElementById(id)) setActiveNav(id);
}

const moltenVertexShader = [
  '#version 300 es',
  'in vec2 position;',
  'void main() {',
  '  gl_Position = vec4(position, 0.0, 1.0);',
  '}',
].join('\n');

const moltenFragmentShader = [
  '#version 300 es',
  'precision highp float;',
  'uniform vec2 iResolution;',
  'uniform float iTime;',
  'uniform float uSpeed;',
  'uniform float uScale;',
  'uniform float uGlow;',
  'uniform float uCoreSize;',
  'uniform float uSwirl;',
  'uniform float uFold;',
  'uniform float uBlackPoint;',
  'uniform float uBrightness;',
  'uniform float uGrainIntensity;',
  'uniform float uOpacity;',
  'uniform vec2 uMouse;',
  'uniform float uMouseStrength;',
  'uniform vec3 uColor1;',
  'uniform vec3 uColor2;',
  'uniform vec3 uColor3;',
  'out vec4 fragColor;',
  '',
  'float hash(vec2 p) {',
  '  return fract(sin(dot(p, vec2(12.9898, 78.233))) * 43758.5453);',
  '}',
  '',
  'void main() {',
  '  float time = iTime * uSpeed;',
  '  vec2 p = uScale * ((gl_FragCoord.xy - 0.5 * iResolution.xy) / iResolution.y) - 0.5;',
  '  p += (uMouse - 0.5) * uMouseStrength * 2.0;',
  '  vec2 i = p;',
  '  float r = length(p + vec2(sin(time), sin(time * 0.3 + 5.0)) * 0.5);',
  '  float d = length(p);',
  '  float rot = d + time + p.x * uSwirl;',
  '  float cosRot = cos(rot);',
  '  mat2 warp = mat2(cos(rot - sin(time / 5.0)), sin(rot), -sin(cosRot - time), cosRot) * uFold;',
  '  float glowCore = uGlow * uCoreSize;',
  '  float c = 0.0;',
  '',
  '  for (float n = 0.0; n < 8.0; n++) {',
  '    if (n >= 4.0) break;',
  '    p *= warp;',
  '    float t = r - time / (n + 3.0);',
  '    i -= p + vec2(cos(t - i.x - r) + sin(t + i.y), sin(t - i.y) + cos(t + i.x) + r);',
  '    c += glowCore / length(vec2(sin(i.x + t), cos(i.y + t)));',
  '  }',
  '',
  '  c /= 6.0;',
  '  float intensity = max(c - uBlackPoint, 0.0) * uBrightness;',
  '  float g = clamp(intensity, 0.0, 1.0);',
  '  vec3 col = mix(uColor1, uColor2, smoothstep(0.0, 0.5, g));',
  '  col = mix(col, uColor3, smoothstep(0.5, 1.0, g));',
  '  float grain = (hash(gl_FragCoord.xy + iTime) - 0.5) * uGrainIntensity;',
  '  float alpha = clamp(g + grain, 0.0, 1.0) * uOpacity;',
  '  fragColor = vec4(col * alpha, alpha);',
  '}',
].join('\n');

function setupMoltenMetalBackground() {
  const canvas = document.querySelector('[data-molten-metal]');
  if (!canvas) return;

  const gl = canvas.getContext('webgl2', {
    alpha: true,
    antialias: false,
    premultipliedAlpha: true,
  });

  if (!gl) {
    canvas.classList.add('molten-fallback');
    return;
  }

  const compileShader = (type, source) => {
    const shader = gl.createShader(type);
    gl.shaderSource(shader, source);
    gl.compileShader(shader);
    if (!gl.getShaderParameter(shader, gl.COMPILE_STATUS)) {
      console.warn('Molten Metal shader compile failed:', gl.getShaderInfoLog(shader));
      gl.deleteShader(shader);
      return null;
    }
    return shader;
  };

  const vertexShader = compileShader(gl.VERTEX_SHADER, moltenVertexShader);
  const fragmentShader = compileShader(gl.FRAGMENT_SHADER, moltenFragmentShader);
  if (!vertexShader || !fragmentShader) {
    canvas.classList.add('molten-fallback');
    return;
  }

  const program = gl.createProgram();
  gl.attachShader(program, vertexShader);
  gl.attachShader(program, fragmentShader);
  gl.linkProgram(program);
  gl.deleteShader(vertexShader);
  gl.deleteShader(fragmentShader);

  if (!gl.getProgramParameter(program, gl.LINK_STATUS)) {
    console.warn('Molten Metal shader link failed:', gl.getProgramInfoLog(program));
    canvas.classList.add('molten-fallback');
    return;
  }

  const buffer = gl.createBuffer();
  gl.bindBuffer(gl.ARRAY_BUFFER, buffer);
  gl.bufferData(gl.ARRAY_BUFFER, new Float32Array([-1, -1, 3, -1, -1, 3]), gl.STATIC_DRAW);

  const position = gl.getAttribLocation(program, 'position');
  const uniforms = {
    resolution: gl.getUniformLocation(program, 'iResolution'),
    time: gl.getUniformLocation(program, 'iTime'),
    speed: gl.getUniformLocation(program, 'uSpeed'),
    scale: gl.getUniformLocation(program, 'uScale'),
    glow: gl.getUniformLocation(program, 'uGlow'),
    coreSize: gl.getUniformLocation(program, 'uCoreSize'),
    swirl: gl.getUniformLocation(program, 'uSwirl'),
    fold: gl.getUniformLocation(program, 'uFold'),
    blackPoint: gl.getUniformLocation(program, 'uBlackPoint'),
    brightness: gl.getUniformLocation(program, 'uBrightness'),
    grainIntensity: gl.getUniformLocation(program, 'uGrainIntensity'),
    opacity: gl.getUniformLocation(program, 'uOpacity'),
    mouse: gl.getUniformLocation(program, 'uMouse'),
    mouseStrength: gl.getUniformLocation(program, 'uMouseStrength'),
    color1: gl.getUniformLocation(program, 'uColor1'),
    color2: gl.getUniformLocation(program, 'uColor2'),
    color3: gl.getUniformLocation(program, 'uColor3'),
  };

  const hexToRgb = hex => {
    const value = hex.replace('#', '');
    return [
      parseInt(value.slice(0, 2), 16) / 255,
      parseInt(value.slice(2, 4), 16) / 255,
      parseInt(value.slice(4, 6), 16) / 255,
    ];
  };

  const mouse = [0.5, 0.5];
  const render = time => {
    gl.useProgram(program);
    gl.bindBuffer(gl.ARRAY_BUFFER, buffer);
    gl.enableVertexAttribArray(position);
    gl.vertexAttribPointer(position, 2, gl.FLOAT, false, 0, 0);
    gl.uniform1f(uniforms.time, time * 0.001);
    gl.uniform1f(uniforms.speed, 0.34);
    gl.uniform1f(uniforms.scale, 4.2);
    gl.uniform1f(uniforms.glow, 1.75);
    gl.uniform1f(uniforms.coreSize, 0.11);
    gl.uniform1f(uniforms.swirl, 1.1);
    gl.uniform1f(uniforms.fold, -0.2);
    gl.uniform1f(uniforms.blackPoint, 0.055);
    gl.uniform1f(uniforms.brightness, 1.35);
    gl.uniform1f(uniforms.grainIntensity, 0.035);
    gl.uniform1f(uniforms.opacity, 0.82);
    gl.uniform2f(uniforms.mouse, mouse[0], mouse[1]);
    gl.uniform1f(uniforms.mouseStrength, 0.2);
    gl.uniform3fv(uniforms.color1, hexToRgb('#141019'));
    gl.uniform3fv(uniforms.color2, hexToRgb('#2b2235'));
    gl.uniform3fv(uniforms.color3, hexToRgb('#a78bfa'));
    gl.drawArrays(gl.TRIANGLES, 0, 3);
  };

  const resize = () => {
    const rect = canvas.getBoundingClientRect();
    const dpr = Math.min(window.devicePixelRatio || 1, 1.5);
    const width = Math.max(1, Math.floor(rect.width * dpr));
    const height = Math.max(1, Math.floor(rect.height * dpr));
    if (canvas.width === width && canvas.height === height) return;
    canvas.width = width;
    canvas.height = height;
    gl.viewport(0, 0, width, height);
    gl.useProgram(program);
    gl.uniform2f(uniforms.resolution, width, height);
    render(0);
  };

  const setMouse = event => {
    const rect = canvas.getBoundingClientRect();
    mouse[0] = (event.clientX - rect.left) / rect.width;
    mouse[1] = 1 - (event.clientY - rect.top) / rect.height;
  };
  const resetMouse = () => {
    mouse[0] = 0.5;
    mouse[1] = 0.5;
  };

  canvas.addEventListener('pointermove', setMouse, { passive: true });
  canvas.addEventListener('pointerleave', resetMouse, { passive: true });
  window.addEventListener('resize', resize, { passive: true });
  resize();

  let animationFrame = 0;
  let visible = true;
  let pageVisible = !document.hidden;

  const stop = () => {
    if (!animationFrame) return;
    window.cancelAnimationFrame(animationFrame);
    animationFrame = 0;
  };

  const start = () => {
    if (reducedMotion || !visible || !pageVisible || animationFrame) return;
    animationFrame = window.requestAnimationFrame(time => {
      animationFrame = 0;
      render(time);
      start();
    });
  };

  const observer = new IntersectionObserver(([entry]) => {
    visible = entry.isIntersecting;
    if (visible) start();
    else stop();
  }, { threshold: 0 });
  observer.observe(canvas);

  document.addEventListener('visibilitychange', () => {
    pageVisible = !document.hidden;
    if (pageVisible) start();
    else stop();
  });

  if (reducedMotion) render(0);
  else start();
}

let scrollFrame = 0;
function handleScroll() {
  if (scrollFrame) return;
  scrollFrame = window.requestAnimationFrame(() => {
    updateHeaderState();
    updateActiveNav();
    updateSectionDepth();
    scrollFrame = 0;
  });
}

navToggle?.addEventListener('click', () => {
  const isOpen = nav?.classList.toggle('is-open') ?? false;
  navToggle.setAttribute('aria-expanded', String(isOpen));
});

navLinks.forEach(link => link.addEventListener('click', () => {
  const id = link.getAttribute('href')?.slice(1);
  if (id) setActiveNav(id);
  closeNav();
}));

function setupAtmosphereEffects() {
  if (reducedMotion) return;

  const pointerTargets = Array.from(document.querySelectorAll('.hero, .signal-band, .feature-board, .atlas-panels, .cta-card'));
  const glareTargets = Array.from(document.querySelectorAll('.signal-item, .feature-shot, .atlas-card, .playbook-card, .module-card, .detail-evidence'));
  const glassTargets = Array.from(document.querySelectorAll('.hero-console, .feature-shot, .atlas-card-dark, .cta-card'));
  const bounceTargets = Array.from(document.querySelectorAll('.playbook-card, .module-card'));
  const magnets = Array.from(document.querySelectorAll('.header-cta, .button'));
  const shimmerTargets = Array.from(document.querySelectorAll('.hero h1, .hero-accent-line em, .section-heading h2 em, .card-kicker, .signal-intro-kicker'));

  glassTargets.forEach(target => target.classList.add('glass-surface'));
  bounceTargets.forEach(target => target.classList.add('bounce-card'));
  shimmerTargets.forEach(target => target.classList.add('text-shimmer'));

  function bindPointerTarget(element, xProperty, yProperty, activeClass) {
    let frame = 0;
    let queuedEvent = null;

    const update = () => {
      frame = 0;
      if (!queuedEvent) return;
      const event = queuedEvent;
      queuedEvent = null;
      const rect = element.getBoundingClientRect();
      element.style.setProperty(xProperty, `${event.clientX - rect.left}px`);
      element.style.setProperty(yProperty, `${event.clientY - rect.top}px`);
      element.classList.add(activeClass);
    };

    element.addEventListener('pointermove', event => {
      if (event.pointerType === 'touch') return;
      queuedEvent = event;
      if (!frame) frame = window.requestAnimationFrame(update);
    });

    element.addEventListener('pointerleave', () => {
      queuedEvent = null;
      if (frame) window.cancelAnimationFrame(frame);
      frame = 0;
      element.classList.remove(activeClass);
      element.style.removeProperty(xProperty);
      element.style.removeProperty(yProperty);
    });
  }

  pointerTargets.forEach(target => {
    target.classList.add('pointer-surface');
    bindPointerTarget(target, '--spot-x', '--spot-y', 'is-spotlit');
  });

  glareTargets.forEach(target => {
    target.classList.add('glare-surface');
    bindPointerTarget(target, '--glare-x', '--glare-y', 'is-glared');
  });

  magnets.forEach(magnet => {
    let frame = 0;
    let queuedEvent = null;

    const update = () => {
      frame = 0;
      if (!queuedEvent) return;
      const event = queuedEvent;
      queuedEvent = null;
      const rect = magnet.getBoundingClientRect();
      const strength = 0.14;
      magnet.style.setProperty('--magnet-x', `${(event.clientX - (rect.left + rect.width / 2)) * strength}px`);
      magnet.style.setProperty('--magnet-y', `${(event.clientY - (rect.top + rect.height / 2)) * strength}px`);
      magnet.classList.add('is-magnetized');
    };

    magnet.addEventListener('pointermove', event => {
      if (event.pointerType === 'touch') return;
      queuedEvent = event;
      if (!frame) frame = window.requestAnimationFrame(update);
    });

    magnet.addEventListener('pointerleave', () => {
      queuedEvent = null;
      if (frame) window.cancelAnimationFrame(frame);
      frame = 0;
      magnet.classList.remove('is-magnetized');
      magnet.style.removeProperty('--magnet-x');
      magnet.style.removeProperty('--magnet-y');
    });
  });
}

function setupHeroTitleFlow() {
  const hero = document.querySelector('.hero');
  const source = hero?.querySelector('#hero-title');
  const target = hero?.querySelector('.hero-accent-line em');
  if (!hero || !source || !target) return;

  const transfer = document.createElement('span');
  transfer.className = 'hero-title-transfer';
  transfer.setAttribute('aria-hidden', 'true');
  hero.append(transfer);

  const updateTransferPosition = () => {
    const heroRect = hero.getBoundingClientRect();
    const sourceRect = source.getBoundingClientRect();
    const targetRect = target.getBoundingClientRect();
    const startX = sourceRect.left + sourceRect.width * 0.8 - heroRect.left;
    const startY = sourceRect.top + sourceRect.height * 0.54 - heroRect.top;
    const endX = targetRect.left + targetRect.width * 0.22 - heroRect.left;
    const endY = targetRect.top + targetRect.height * 0.52 - heroRect.top;

    transfer.style.setProperty('--transfer-start-x', `${startX}px`);
    transfer.style.setProperty('--transfer-start-y', `${startY}px`);
    transfer.style.setProperty('--transfer-end-x', `${endX}px`);
    transfer.style.setProperty('--transfer-end-y', `${endY}px`);
    transfer.style.setProperty('--transfer-angle', `${Math.atan2(endY - startY, endX - startX)}rad`);
  };

  updateTransferPosition();
  if (reducedMotion) return;

  hero.classList.add('has-hero-title-flow');
  window.setTimeout(updateTransferPosition, 900);
  window.addEventListener('resize', updateTransferPosition, { passive: true });
}

if ('IntersectionObserver' in window && !reducedMotion) {
  const revealObserver = new IntersectionObserver(
    entries => {
      entries.forEach(entry => {
        if (!entry.isIntersecting) return;
        entry.target.classList.add('is-visible');
        revealObserver.unobserve(entry.target);
      });
    },
    { rootMargin: '0px 0px -10% 0px', threshold: 0.08 }
  );

  revealItems.forEach(item => revealObserver.observe(item));
} else {
  revealItems.forEach(item => item.classList.add('is-visible'));
}

// If the page is opened directly at an in-page anchor, the browser may place
// the target in view before IntersectionObserver receives its first callback.
// Reveal the target and its reveal ancestors immediately so deep links never
// land on a visually empty section.
const initialAnchorId = window.location.hash.replace(/^#/, '');
if (initialAnchorId) {
  const initialAnchor = document.getElementById(decodeURIComponent(initialAnchorId));
  let revealAncestor = initialAnchor;
  while (revealAncestor && revealAncestor !== document.body) {
    if (revealAncestor.classList.contains('reveal')) revealAncestor.classList.add('is-visible');
    revealAncestor = revealAncestor.parentElement;
  }
  initialAnchor?.querySelectorAll('.reveal').forEach(item => item.classList.add('is-visible'));
}

function setupTabs() {
  const tabGroups = [
    {
      tabs: Array.from(document.querySelectorAll('[data-feature-tab]')),
      panels: Array.from(document.querySelectorAll('[data-feature-panel]')),
      tabKey: 'featureTab',
      panelKey: 'featurePanel',
    },
    {
      tabs: Array.from(document.querySelectorAll('[data-atlas-tab]')),
      panels: Array.from(document.querySelectorAll('[data-atlas-panel]')),
      tabKey: 'atlasTab',
      panelKey: 'atlasPanel',
    },
  ];

  tabGroups.forEach(({ tabs, panels, tabKey, panelKey }) => {
    if (!tabs.length || !panels.length) return;

    const setActive = index => {
      tabs.forEach(tab => {
        const active = Number(tab.dataset[tabKey]) === index;
        tab.classList.toggle('is-active', active);
        tab.setAttribute('aria-selected', String(active));
      });

      panels.forEach(panel => {
        const active = Number(panel.dataset[panelKey]) === index;
        panel.classList.toggle('is-active', active);
        panel.hidden = !active;
      });
    };

    tabs.forEach((tab, index) => {
      tab.addEventListener('click', () => setActive(Number(tab.dataset[tabKey])));
      tab.addEventListener('keydown', event => {
        if (event.key !== 'ArrowRight' && event.key !== 'ArrowLeft') return;
        event.preventDefault();
        const nextIndex = event.key === 'ArrowRight' ? (index + 1) % tabs.length : (index - 1 + tabs.length) % tabs.length;
        tabs[nextIndex].focus();
        setActive(nextIndex);
      });
    });

    setActive(0);
  });
}

function setupFunctionIndex() {
  const details = Array.from(document.querySelectorAll('.function-detail'));
  const evidence = [
    { image: 'home.png', label: '工作台 / 实际界面', title: '运行态、统计和快捷入口先汇总在一起。', tags: ['实例统计', '系统事件', '快捷操作'] },
    { image: 'instances.png', label: '实例列表 / 实际界面', title: '每个浏览器环境都能被创建、启动、复制和流转。', tags: ['实例生命周期', 'Launch Code', '批量操作'] },
    { image: 'fingerprint.png', label: '指纹 / 实际界面', title: '把身份、设备和启动参数放在同一个配置面板。', tags: ['设备画像', '一致性', '启动参数'] },
    { image: 'proxy-pool.png', label: '代理池 / 实际界面', title: '节点、检测、订阅和资源状态在同一张网络工作台里。', tags: ['订阅导入', '健康检查', '测速'] },
    { image: 'proxy-routes.png', label: '代理路由 / 实际界面', title: '按域名、路径和出口策略组织真实请求链路。', tags: ['分流', '链式代理', '故障切换'] },
    { image: 'cores.png', label: '浏览器内核 / 实际界面', title: '浏览器内核、版本和全局启动参数都有明确归属。', tags: ['版本管理', '下载', '全局设置'] },
    { image: 'extensions.png', label: '扩展 / 实际界面', title: '插件从导入到实例绑定，都能被持续管理。', tags: ['ZIP / CRX', '绑定', '同步'] },
    { image: 'organization.png', label: '组织管理 / 实际界面', title: '标签、分组和默认内容帮助大量实例保持秩序。', tags: ['标签', '分组', '默认内容'] },
    { image: 'window-sync.png', label: '窗口联动 / 实际界面', title: '多个实例可以同时接收输入、URL 和布局控制。', tags: ['同步输入', '批量 URL', '多窗口'] },
    { image: 'rpa.png', label: 'RPA / 实际界面', title: '从流程入口到任务运行，自动化有自己的工作区。', tags: ['流程设计', '节点图', '可运行'] },
    { image: 'rpa.png', label: 'SELECTOR / RPA SCREEN', title: '选择器、录制器和页面交互共同支撑稳定节点。', tags: ['CSS / XPath', '录制', '元素拾取'] },
    { image: 'rpa.png', label: 'WEB ACTIONS / RPA SCREEN', title: '表单、等待、上传、悬停和 iframe 都能被编排。', tags: ['表单', '等待', '上传'] },
    { image: 'rpa.png', label: 'DATA / RPA SCREEN', title: '从列表到详情，再到 JSON、CSV 和文本结果。', tags: ['列表采集', 'SKU', '导出'] },
    { image: 'automation.png', label: '自动化 / 实际界面', title: '邮箱、验证码和外部接口都留下可追踪的调用状态。', tags: ['邮箱池', '验证码', '脱敏'] },
    { image: 'rpa.png', label: 'WORKFLOW ASSET / RPA SCREEN', title: '包、版本、计划和存储让一次流程可以长期复用。', tags: ['版本', '计划', '存储'] },
    { image: 'rpa-scheduler.png', label: '任务中心 / 实际界面', title: '立即执行、定时和循环任务都可以统一调度。', tags: ['任务', '调度', '并发'] },
    { image: 'logs.png', label: '运行历史 / 实际界面', title: '节点输入、输出、耗时和失败阶段都能回看。', tags: ['运行历史', '错误阶段', '重试'] },
    { image: 'automation.png', label: 'Launch Server / 实际界面', title: '外部脚本通过固定端口接管实例和 CDP。', tags: ['HTTP + JSON', 'Launch Code', 'CDP'] },
    { image: 'backup.png', label: '同步与备份 / 实际界面', title: '配置、Cookie、云端备份和实例分享都可控地流转。', tags: ['备份', '加密', '分享'] },
    { image: 'settings.png', label: '维护 / 实际界面', title: '主题、日志、配置恢复和更新策略集中维护。', tags: ['主题', '日志', '更新'] },
    { image: 'launch-api.png', label: '基础能力 / 实际界面', title: '本地优先、桌面运行、CDP 接入和边界责任都清晰可见。', tags: ['React', 'Wails', 'Go / CDP'] },
  ];

  details.forEach((detail, index) => {
    const body = detail.querySelector('.detail-body');
    const item = evidence[index];
    if (!body || !item) return;

    const panel = document.createElement('div');
    panel.className = 'detail-evidence';
    panel.innerHTML = `
      <div class="detail-evidence-copy">
        <small>${item.label}</small>
        <strong>${item.title}</strong>
        <div class="detail-evidence-tags">${item.tags.map(tag => `<span>${tag}</span>`).join('')}</div>
      </div>
      <figure class="detail-evidence-media">
        <img src="assets/images/${item.image}" alt="${item.title}" loading="lazy" decoding="async" />
      </figure>
    `;
    body.prepend(panel);
  });
}

function setupThemeShowcase() {
  const showcase = document.querySelector('[data-theme-showcase]');
  const choices = Array.from(showcase?.querySelectorAll('[data-theme-choice]') || []);
  const name = showcase?.querySelector('[data-theme-name]');
  const description = showcase?.querySelector('[data-theme-current-description]');
  const status = showcase?.querySelector('[data-theme-status]');
  if (!showcase || !choices.length || !name || !description || !status) return;

  const variables = [
    ['bg', 'themeBg'],
    ['surface', 'themeSurface'],
    ['ink', 'themeInk'],
    ['accent', 'themeAccent'],
    ['soft', 'themeSoft'],
  ];
  const autoPlayDelay = 3000;
  const manualResumeDelay = 6500;
  let activeIndex = 0;
  let autoPlayTimer = 0;
  let resumeTimer = 0;
  let interactionPaused = false;

  choices.forEach(choice => {
    variables.forEach(([key, datasetKey]) => {
      const value = choice.dataset[datasetKey];
      if (value) choice.style.setProperty(`--choice-${key}`, value);
    });
  });

  function selectChoice(choice, { automatic = false } = {}) {
    variables.forEach(([key, datasetKey]) => {
      showcase.style.setProperty(`--showcase-${key}`, choice.dataset[datasetKey]);
    });

    activeIndex = Math.max(0, choices.indexOf(choice));
    showcase.dataset.activeTheme = choice.dataset.themeChoice || '';
    name.textContent = choice.dataset.themeLabel || '';
    description.textContent = choice.dataset.themeDescription || '';
    status.textContent = automatic ? '自动切换' : '已选择';

    choices.forEach(item => {
      const active = item === choice;
      item.classList.toggle('is-active', active);
      item.setAttribute('aria-pressed', String(active));
    });
  }

  function scheduleAutoPlay(delay = autoPlayDelay) {
    window.clearTimeout(autoPlayTimer);
    if (reducedMotion || interactionPaused || document.hidden) return;

    autoPlayTimer = window.setTimeout(() => {
      const nextIndex = (activeIndex + 1) % choices.length;
      selectChoice(choices[nextIndex], { automatic: true });
      scheduleAutoPlay();
    }, delay);
  }

  function pauseAutoPlay() {
    interactionPaused = true;
    window.clearTimeout(autoPlayTimer);
    window.clearTimeout(resumeTimer);
  }

  function resumeAutoPlay(delay = autoPlayDelay) {
    interactionPaused = false;
    window.clearTimeout(resumeTimer);
    scheduleAutoPlay(delay);
  }

  choices.forEach(choice => choice.addEventListener('click', () => {
    selectChoice(choice);
    window.clearTimeout(autoPlayTimer);
    window.clearTimeout(resumeTimer);
    resumeTimer = window.setTimeout(() => resumeAutoPlay(), manualResumeDelay);
  }));

  showcase.addEventListener('mouseenter', pauseAutoPlay);
  showcase.addEventListener('mouseleave', () => resumeAutoPlay(1200));
  showcase.addEventListener('focusin', pauseAutoPlay);
  showcase.addEventListener('focusout', event => {
    if (!showcase.contains(event.relatedTarget)) resumeAutoPlay(1200);
  });
  document.addEventListener('visibilitychange', () => {
    if (document.hidden) {
      window.clearTimeout(autoPlayTimer);
      return;
    }
    scheduleAutoPlay(1200);
  });

  const initialChoice = choices.find(choice => choice.classList.contains('is-active')) || choices[0];
  selectChoice(initialChoice, { automatic: !reducedMotion });
  scheduleAutoPlay();
}

function setupDownloadChoices() {
  const choices = Array.from(document.querySelectorAll('details[data-download-choice]'));
  if (!choices.length) return;

  const openChoiceFromHash = (scroll = false) => {
    const id = window.location.hash.replace(/^#/, '');
    const target = choices.find(choice => choice.id === id);
    if (!target) return;

    choices.forEach(choice => {
      choice.open = choice === target;
    });
    if (scroll) {
      window.setTimeout(() => target.scrollIntoView({ behavior: 'smooth', block: 'center' }), 0);
    }
  };

  choices.forEach(choice => {
    let closeTimer = 0;

    choice.addEventListener('toggle', () => {
      if (!choice.open) return;
      choices.forEach(other => {
        if (other !== choice) other.open = false;
      });
    });

    choice.addEventListener('mouseenter', () => {
      window.clearTimeout(closeTimer);
      choice.open = true;
    });

    choice.addEventListener('mouseleave', () => {
      window.clearTimeout(closeTimer);
      closeTimer = window.setTimeout(() => {
        if (!choice.matches(':focus-within')) choice.open = false;
      }, 180);
    });

    choice.addEventListener('focusin', () => {
      window.clearTimeout(closeTimer);
      choice.open = true;
    });

    choice.addEventListener('focusout', event => {
      if (choice.contains(event.relatedTarget)) return;
      window.clearTimeout(closeTimer);
      closeTimer = window.setTimeout(() => {
        if (!choice.matches(':hover')) choice.open = false;
      }, 120);
    });
  });

  window.addEventListener('hashchange', () => openChoiceFromHash(true));
  openChoiceFromHash(false);
}

function setupImageLightbox() {
  const lightbox = document.querySelector('[data-image-lightbox]');
  const preview = lightbox?.querySelector('[data-lightbox-image]');
  const caption = lightbox?.querySelector('[data-lightbox-caption]');
  const closeButtons = Array.from(lightbox?.querySelectorAll('[data-lightbox-close]') || []);
  const images = Array.from(document.querySelectorAll('main img, .site-footer img')).filter(image => !image.closest('.brand'));
  if (!lightbox || !preview || !caption || !images.length) return;

  let previousFocus = null;

  function close() {
    if (lightbox.hidden) return;
    lightbox.hidden = true;
    document.body.classList.remove('has-lightbox');
    preview.removeAttribute('src');
    if (previousFocus instanceof HTMLElement) previousFocus.focus();
  }

  function open(image) {
    previousFocus = document.activeElement;
    preview.src = image.currentSrc || image.src;
    preview.alt = image.alt || 'Trace Browser 图片预览';
    caption.textContent = image.alt || 'Trace Browser 图片预览';
    lightbox.hidden = false;
    document.body.classList.add('has-lightbox');
    lightbox.querySelector('.image-lightbox-close')?.focus();
  }

  images.forEach(image => {
    image.classList.add('lightbox-trigger');
    image.addEventListener('click', event => {
      event.preventDefault();
      event.stopPropagation();
      open(image);
    });

    image.tabIndex = 0;
    image.setAttribute('role', 'button');
    image.setAttribute('aria-label', `放大查看：${image.alt || 'Trace Browser 图片'}`);
    image.addEventListener('keydown', event => {
      if (event.key !== 'Enter' && event.key !== ' ') return;
      event.preventDefault();
      open(image);
    });
  });

  closeButtons.forEach(button => button.addEventListener('click', close));
  document.addEventListener('keydown', event => {
    if (event.key === 'Escape' && !lightbox.hidden) close();
  });
}

window.addEventListener('scroll', handleScroll, { passive: true });
window.addEventListener('resize', handleScroll, { passive: true });
window.addEventListener('hashchange', syncHashNav);

updateHeaderState();
updateActiveNav();
updateSectionDepth();
syncHashNav();
window.setTimeout(updateActiveNav, 300);
window.setTimeout(updateActiveNav, 1200);
window.setTimeout(syncHashNav, 1600);
setupTabs();
setupFunctionIndex();
setupThemeShowcase();
setupDownloadChoices();
setupMoltenMetalBackground();
setupHeroTitleFlow();
setupAtmosphereEffects();
setupImageLightbox();
