;(async function(){
  let currentReversion = 0;
  await fetch('/reversion.test').then(res => res.json()).then(res => {
    if(res.reversion !== 0){
      currentReversion = res.reversion;
    }
  }).catch(e => console.error);

  let interval;
  async function checkForUpdate(){
    await fetch('/reversion.test').then(res => res.json()).then(res => {
      if(res.reversion !== currentReversion){
        window.location.reload();
      }
    }).catch(e => {
      clearInterval(interval);
      setTimeout(function(){
        interval = setInterval(checkForUpdate, 300);
      }, 10000);
    });
  }

  interval = setInterval(checkForUpdate, 300);

  console.log('Hot Reload Enabled!');
})();
