package main

// dragHandlerJS is injected into the webview on DomReady.
// It bypasses Wails' built-in CSS-variable-based drag detection (which uses
// getComputedStyle and has timing/inheritance issues with dynamic SPA content)
// and instead uses robust DOM-traversal via el.closest() plus a Y-position
// fallback for the top 40px of a layout container, including its children.
// The "drag"
// message is sent directly through the WKWebView script-message bridge,
// which the native Objective-C handler in WailsContext.m converts to
// [NSWindow performWindowDragWithEvent:].
const dragHandlerJS = `(function(){
if(window.__wkDragBound)return;
window.__wkDragBound=true;
document.documentElement.classList.add('wails-desktop');

// Disable rubber-band overscroll that reveals the dark window background
document.documentElement.style.overscrollBehavior='none';
document.body.style.overscrollBehavior='none';

if(window.wails&&window.wails.flags){
  window.wails.flags.cssDragProperty='__disabled__';
  window.wails.flags.cssDragValue='__never__';
}

// Prevent native text/image drag-out to fix the "selected and dragged away" issue
window.addEventListener('dragstart', function(e){
  e.preventDefault();
}, true);

var TITLEBAR_H=40;

// We specifically look for Wails' inline style attributes injected by Vue,
// and custom drag classes, avoiding generic headers like .section-header.
// .chat-topbar is the window top on the conversation page.
var dragSel='.logo_row,.menu_top,.chat-topbar,.drag-region,[data-wails-drag],' +
  '[style*="--wails-draggable: drag"],[style*="--wails-draggable:drag"]';

var noDragSel='button,a,input,select,textarea,[role="button"],' +
  '.t-button,.t-input,.t-select,.t-textarea,' +
  '.header-actions,.header-action-btn,.sidebar-toggle,.logo_box,' +
  '.close-btn,.menu_item,.submenu,.submenu_item,.menu_bottom,' +
  '.t-popup,.t-dropdown,.t-tooltip,.t-dialog,[data-no-drag],' +
  '[style*="--wails-draggable: no-drag"],[style*="--wails-draggable:no-drag"]';

var layoutClasses=['main','chat','dialogue-wrap','kb-list-container','kb-list-content',
  'agent-list-container','agent-list-content','org-list-container','org-list-content','aside_box',
  'ks-container','ks-content','settings-overlay','knowledge-layout',
  'faq-manager-wrapper','login-layout'];

function sendDrag(){
  try{window.webkit.messageHandlers.external.postMessage('drag')}
  catch(_){try{window.WailsInvoke('drag')}catch(e){}}
}

function isLayoutEl(el){
  for(var i=0;i<layoutClasses.length;i++){
    if(el.classList.contains(layoutClasses[i]))return true;
  }
  var tag=el.tagName;
  return tag==='BODY'||tag==='HTML';
}

function inTitlebar(el,y){
  if(y>TITLEBAR_H)return false;
  var node=el;
  while(node&&node instanceof Element){
    if(isLayoutEl(node))return true;
    node=node.parentElement;
  }
  return false;
}

function shouldDrag(el,y){
  if(!(el instanceof Element))return false;
  if(el.closest(noDragSel))return false;
  if(el.closest(dragSel))return true;
  if(inTitlebar(el,y))return true;
  return false;
}

window.addEventListener('mousedown',function(e){
  var target=e.target;
  if(target&&target.nodeType===Node.TEXT_NODE){
    target=target.parentElement;
  }
  if(e.button!==0||e.detail!==1)return;
  if(!shouldDrag(target,e.clientY))return;
  // The session title renames on double-click. Wait until the pointer
  // actually moves so that click still lands; other top-bar hits drag now.
  var defer=target.closest&&target.closest('.chat-header__title');
  if(!defer){
    e.preventDefault();
    sendDrag();
    return;
  }
  var x=e.clientX,y=e.clientY;
  function move(ev){
    if(Math.abs(ev.clientX-x)<4&&Math.abs(ev.clientY-y)<4)return;
    cleanup();
    sendDrag();
  }
  function up(){cleanup();}
  function cleanup(){
    window.removeEventListener('mousemove',move,true);
    window.removeEventListener('mouseup',up,true);
  }
  window.addEventListener('mousemove',move,true);
  window.addEventListener('mouseup',up,true);
},true);

// Intercept external link clicks and window.open so they open in the system browser
document.addEventListener('click',function(e){
  var el=e.target;
  while(el&&el.tagName!=='A')el=el.parentElement;
  if(!el||!el.href)return;
  var href=el.href;
  if(el.hasAttribute('download'))return;
  try{var targetURL=new URL(href);if(targetURL.origin===window.location.origin&&targetURL.host===window.location.host)return;}catch(_){}
  if(href.indexOf('http://')===0||href.indexOf('https://')===0){
    if(window.runtime&&window.runtime.BrowserOpenURL){
      e.preventDefault();
      e.stopPropagation();
      window.runtime.BrowserOpenURL(href);
    }
  }
},true);

var origOpen=window.open;
window.open=function(url){
  try{var u=new URL(url,window.location.href);if(u.host===window.location.host&&u.protocol===window.location.protocol)return origOpen.apply(window,arguments);}catch(_){}
  if(url&&(typeof url==='string')&&(url.indexOf('http://')===0||url.indexOf('https://')===0)){
    if(window.runtime&&window.runtime.BrowserOpenURL){
      window.runtime.BrowserOpenURL(url);
      return null;
    }
  }
  return origOpen.apply(window,arguments);
};
})();`

// wailsThemeSyncJS：与 index.html 首屏一致，在 DomReady 再跑一遍，覆盖 Ctrl+R 后 runtime 就绪时机
const wailsThemeSyncJS = `(function(){try{var t=localStorage.getItem('OPC_theme')||localStorage.getItem('WeKnora_theme')||'light';if(t==='system')t=window.matchMedia('(prefers-color-scheme: dark)').matches?'dark':'light';var bg=t==='dark'?'#181818':'#eee';document.documentElement.setAttribute('theme-mode',t);document.documentElement.style.background=bg;document.documentElement.style.minHeight='100%';document.documentElement.style.colorScheme=t==='dark'?'dark':'light';if(document.body){document.body.style.background=bg;document.body.style.minHeight='100%';}var w=window.runtime;if(!w)return;if(t==='dark'){if(w.WindowSetDarkTheme)w.WindowSetDarkTheme();if(w.WindowSetBackgroundColour)w.WindowSetBackgroundColour(24,24,24,255);}else{if(w.WindowSetLightTheme)w.WindowSetLightTheme();if(w.WindowSetBackgroundColour)w.WindowSetBackgroundColour(238,238,238,255);}}catch(e){}})()`
