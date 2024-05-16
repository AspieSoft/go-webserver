/*! Smart Theme v0.1.1 | MIT License | github.com/AspieSoft/smart-theme */

function sleep(ms){return new Promise(resolve => setTimeout(resolve, ms));}

async function onReady(cb){
  if(document.readyState !== 'loading'){
    cb();
  }else{
    let ranCB = false;
    document.addEventListener('DOMContentLoaded', function () {
      ranCB = true;
      cb();
    });

    sleep(100);

    if(!ranCB){
      let loops = 10;
      while(loops > 0){
        if(document.readyState !== 'loading'){
          cb();
          return;
        }

        loops--;
        sleep(250);
      }
    }
  }
}

onReady(async function(){
  const body = document.body || document.querySelector('body');
  const headerTop = document.querySelector('header .header-top');
  const headerTopNav = document.querySelector('header .header-top nav.top-nav > ul');

  if(body && window.location.pathname === '/'){
    body.classList.add('home');
  }


  function onInterval(){
    document.querySelectorAll('.widget > *, .sidebar > *').forEach(function(elm) {
      if(elm.clientHeight < window.innerHeight - 150) {
        elm.classList.add('widget-smaller-than-vh');
      }else{
        elm.classList.remove('widget-smaller-than-vh');
      }
    });

    if(headerTop){
      headerTop.querySelectorAll('a.nav-btn:not([nav-btn-ready]), li.nav-btn a:not([nav-btn-ready])').forEach(function(elm) {
        elm.setAttribute('nav-btn-ready', '');
  
        elm.addEventListener('click', function(e){
          e.preventDefault();
          headerNavBtnClick();
        });
      });
    }
  }
  onInterval();
  setInterval(onInterval, 1000);

  function headerNavBtnClick(){
    if(window.innerWidth >= 1400){
      headerTop.classList.remove('nav-open');
      headerTop.classList.toggle('nav-closed');
    }else{
      headerTop.classList.remove('nav-closed');
      headerTop.classList.toggle('nav-open');
    }
  }


  function onResize(){
    if(headerTop && headerTopNav){
      let headerRect = headerTop.getBoundingClientRect();
      let hW = Math.round(headerRect.width);

      if(headerTop.scrollWidth > hW){
        let allHidden = true;

        for(let i = headerTopNav.children.length - 1; i >= 0; i--){
          headerTopNav.children[i].style['display'] = 'none';
          headerRect = headerTop.getBoundingClientRect();
          if(headerTop.scrollWidth <= hW){
            allHidden = false;
            break;
          }
        }

        if(allHidden){
          headerTop.querySelectorAll('nav:not(.top-nav) > ul').forEach(function(elm){
            if(!allHidden){
              return;
            }

            if(headerTop.scrollWidth > hW){
              for(let i = 0; i < elm.children.length; i++){
                elm.children[i].style['display'] = 'none';
                headerRect = headerTop.getBoundingClientRect();
                if(headerTop.scrollWidth <= hW){
                  allHidden = false;
                  break;
                }
              }
            }
          });
        }
      }else{
        let allVisible = true;

        headerTop.querySelectorAll('nav:not(.top-nav) > ul').forEach(function(elm){
          if(!allVisible){
            return;
          }

          if(headerTop.scrollWidth <= hW){
            for(let i = elm.children.length - 1; i >= 0; i--){
              elm.children[i].style['display'] = '';
              headerRect = headerTop.getBoundingClientRect();
              if(headerTop.scrollWidth > hW){
                elm.children[i].style['display'] = 'none';
                allVisible = false;
                break;
              }
            }
          }
        });

        if(allVisible){
          for(let i = 0; i < headerTopNav.children.length; i++){
            headerTopNav.children[i].style['display'] = '';
            headerRect = headerTop.getBoundingClientRect();
            if(headerTop.scrollWidth > hW){
              headerTopNav.children[i].style['display'] = 'none';
              break;
            }
          }
        }
      }
    }

    if(headerTop && !headerTop.classList.contains('nav-desktop') && window.innerWidth >= 800){
      headerTop.classList.remove('nav-open');
    }
  }
  onResize();
  window.addEventListener('resize', onResize, {passive: true});


  //todo: add other imidiatelly visible header elements to list
  document.querySelectorAll('html, header .header-img, header .header-top').forEach(function(elm){
    const css = window.getComputedStyle(elm);
    let src = css.getPropertyValue('--img');
    if(src.startsWith('url')){
      src = src.replace(/^\s*[\w_-]+\s*\(\s*(["'`])(.*?)\1\s*\)\s*$/, '$2');

      if(!src.match(/\.[0-9]+p\.([\w_-]+)$/)){
        return;
      }

      if(window.innerWidth <= 800){
        src = src.replace(/\.[0-9]+p\.([\w_-]+)$/, '.720p.$1');
      }else if(window.innerWidth <= 2000){
        src = src.replace(/\.[0-9]+p\.([\w_-]+)$/, '.1080p.$1');
      }else{
        src = src.replace(/\.[0-9]+p\.([\w_-]+)$/, '.$1');
      }

      const img = new Image();
      img.onload = function(){
        elm.style.setProperty('--img', `url('${src.replace(/(\\*)([\\'])/g, function(_, s, q){
          if(s.length % 0 === 0){
            return s+'\\'+q;
          }else{
            return s+q;
          }
        })}')`);
      };
      img.onerror = function(){
        if(src.match(/\.[0-9]+p\.([\w_-]+)$/)){
          src = src.replace(/\.[0-9]+p\.([\w_-]+)$/, '.$1');
          img.src = src;
        }
      };
      img.src = src;
    }
  });

});

onReady(async function(){
  function onInterval(){
    document.querySelectorAll('header .header-img, footer').forEach(function(elm){
      if(elm.clientHeight < 300){
        elm.classList.remove('background-size-anim-w', 'background-size-anim-h');
        return;
      }

      if(elm.classList.contains('background-size-anim-w') || elm.classList.contains('background-size-anim-h')){
        return;
      }

      const css = window.getComputedStyle(elm);
      if(typeof css.getPropertyValue('--img-size') !== 'string' || typeof css.getPropertyValue('--img') !== 'string'){
        return;
      }

      if(css.getPropertyValue('--img-size').includes('cover')){
        css.getPropertyValue('--img').replace(/url\s*\(\s*(["'`])(.*?)\1\s*\)/i, function(_, _, url){
          const img = new Image();
          img.src = url;
          img.onload = function(){
            if(img.width !== 0 && img.height !== 0){
              elm.setAttribute('img-width', img.width);
              elm.setAttribute('img-height', img.height);

              elm.style.setProperty('--scale-offset', calculateImgSizeRatioDiff(elm, img.width, img.height) + 'px');
            }
            img.remove();
          }
        });
      }
    });

    document.querySelectorAll('header .header-top').forEach(function(elm) {
      const css = window.getComputedStyle(elm);
      const img = css.getPropertyValue('--img');
      if(img !== '' && img !== 'none' && img.match(/^\s*([\w_-]+-gradient|url)/)){
        elm.style['background-color'] = 'transparent';
        elm.style['animation-name'] = 'scroll-header-top-shadow';
      }else{
        elm.style['background-color'] = '';
        elm.style['animation-name'] = '';
      }
    });
  }
  onInterval();
  setInterval(onInterval, 1000);


  function onResize(){
    document.querySelectorAll('header .header-img, footer').forEach(function(elm){
      if(elm.clientHeight < 300){
        elm.classList.remove('background-size-anim-w', 'background-size-anim-h');
        return;
      }
      
      if(!elm.classList.contains('background-size-anim-w') && !elm.classList.contains('background-size-anim-h')){
        return;
      }
      
      const imgWidth = elm.getAttribute('img-width');
      const imgHeight = elm.getAttribute('img-height');
      if(!imgWidth || !imgHeight || imgWidth === 0 || imgHeight === 0){
        return;
      }

      elm.style.setProperty('--scale-offset', calculateImgSizeRatioDiff(elm, imgWidth, imgHeight) + 'px');
    });
  }
  onResize();
  window.addEventListener('resize', onResize, {passive: true});


  // complex math to calculate the offset to add to a background image size
  // add this number to `background-size: auto 100%` to simulate `cover`
  function calculateImgSizeRatioDiff(elm, imgWidth, imgHeight){
    if(imgWidth / imgHeight < elm.clientWidth / elm.clientHeight){
      elm.classList.add('background-size-anim-w');
      elm.classList.remove('background-size-anim-h');
    }else{
      elm.classList.add('background-size-anim-h');
      elm.classList.remove('background-size-anim-w');
    }

    return (elm.clientWidth / imgWidth) * (imgWidth / imgHeight) * 100;
  }

});
