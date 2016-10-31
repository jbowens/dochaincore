$(document).ready(function() {

  var currentPct = 0;

  function updateProgressBar(pct, extraMax, extraDuration) {
      if (currentPct != pct) {
          currentPct = pct;
          $('#current-progress').stop();
          $('#current-progress').animate({width: pct+'%'}, 1000, 'swing', function() {

            // If there's an extra easing, then animate that too.
            if (extraMax && extraDuration) {
              $('#current-progress').animate({width: extraMax+'%'}, extraDuration);
            }
          });
      }
  }

  function updateUI() {
      $.get("/status/"+window.installID, function(resp) {
          console.log(resp);
          if (resp.status == 'pending auth') {
              $('#status-line').text('Provisioning droplet…');
              updateProgressBar(5);
          } else if (resp.status == 'waiting for ssh') {
              $('#status-line').text('Waiting for SSH…');
              updateProgressBar(10, 45, 45000);
          } else if (resp.status == 'waiting for http') {
              $('#status-line').text('Waiting for HTTP…');
              updateProgressBar(45, 90, 45000);
          } else if (resp.status == 'creating client token') {
              $('#status-line').text('Creating client token…');
              updateProgressBar(95, 99, 2000);
          } else if (resp.status == 'done') {
              $('#status-line').text('Install complete');
              updateProgressBar(100);
              $('#client-token').text(resp.client_token);
              $('#core-url').text('http://'+resp.ip_address+':1999');
              $('#open-dashboard').attr('href', 'http://' + resp.client_token + '@' + resp.ip_address + ':1999/dashboard');
              $('#core-info').css('display', 'block');
          } else {
              $('status-line').text('Install failed: ' + resp.status);
              updateProgressBar(0);
          }

          if (resp.status != 'done') {
              setTimeout(updateUI, 1000);
          }
      });
  }

  updateUI();
});
