(function(){var main=FX.$('main'),hero=FX.$('.hero'),board=FX.$('.board');if(!main||!hero)return;var cv=document.createElement('canvas');cv.className='rad';cv.setAttribute('aria-hidden','true');main.insertBefore(cv,main.firstChild);
function H(){return hero.offsetTop+hero.offsetHeight+30}cv.style.height=H()+'px';addEventListener('resize',function(){cv.style.height=H()+'px'});
var NM=['Firebase','Google Maps','reCAPTCHA','OpenAI','Stripe','YouTube','Fonts','Analytics','Sentry'],ST=['#ff5d5d','#ffa53b','#e07cff','#ff5d5d','#ff5d5d','#ffa53b','#e8d36b','#e07cff','#7d7c78'],bl=[],pings=[];
var rs=(function(s){return function(){s=(s*16807)%2147483647;return s/2147483647}})(11);
for(var i=0;i<NM.length;i++)bl.push({a:rs()*6.283,r:.25+rs()*.7,n:NM[i],c:ST[i],seen:-9});
cv.addEventListener('pointerdown',function(e){var r=cv.getBoundingClientRect();pings.push({x:e.clientX-r.left,y:e.clientY-r.top,t:performance.now()})});
FX.loop(cv,function(c,w,h,t,dt,m){var D=FX.theme()==='dark',I=D?'255,255,255':'0,0,0';c.clearRect(0,0,w,h);var br=board?board.getBoundingClientRect():null,vr=cv.getBoundingClientRect();
 var narrow=w<860,cx=narrow?w*.75:(br?br.left+br.width/2-vr.left:w*.72),cy=narrow?h*.2:(br?br.top+br.height/2-vr.top:h*.5),R=Math.min(narrow?w*.7:w*.42,h*.62);
 /* border line: slanted dashed with checkpoints */
 c.strokeStyle='rgba('+I+',.14)';c.lineWidth=1;c.setLineDash([10,8]);c.lineDashOffset=-t*8;c.beginPath();c.moveTo(0,h*.82);c.bezierCurveTo(w*.3,h*.7,w*.55,h*.95,w,h*.78);c.stroke();c.setLineDash([]);
 /* rings */ [1,.75,.5,.25].forEach(function(k){c.strokeStyle='rgba('+I+','+(.09+k*.05)+')';c.beginPath();c.arc(cx,cy,R*k,0,6.3);c.stroke()});
 c.strokeStyle='rgba('+I+',.1)';c.beginPath();c.moveTo(cx-R,cy);c.lineTo(cx+R,cy);c.moveTo(cx,cy-R);c.lineTo(cx,cy+R);c.stroke();
 for(var q=0;q<72;q++){var a=q*Math.PI/36,l=q%6?4:10;c.beginPath();c.moveTo(cx+Math.cos(a)*R,cy+Math.sin(a)*R);c.lineTo(cx+Math.cos(a)*(R-l),cy+Math.sin(a)*(R-l));c.stroke()}
 /* sweep */ var sa=t*1.1%6.2832;for(var k=0;k<44;k++){var aa=sa-k*.02;c.strokeStyle='rgba('+I+','+(.28*(1-k/44))+')';c.beginPath();c.moveTo(cx,cy);c.lineTo(cx+Math.cos(aa)*R,cy+Math.sin(aa)*R);c.stroke()}
 c.strokeStyle='rgba('+I+',.8)';c.lineWidth=1.5;c.beginPath();c.moveTo(cx,cy);c.lineTo(cx+Math.cos(sa)*R,cy+Math.sin(sa)*R);c.stroke();c.lineWidth=1;
 /* blips */ c.font='700 11px ui-monospace,monospace';bl.forEach(function(b){var d=((sa-b.a)%6.2832+6.2832)%6.2832;if(d<.06&&t>b.seen+1)b.seen=t;var age=t-b.seen;if(age<0||age>5)return;var al=1-age/5,x=cx+Math.cos(b.a)*b.r*R,y=cy+Math.sin(b.a)*b.r*R;
  c.globalAlpha=al*(narrow?.45:1);c.fillStyle=b.c;c.beginPath();c.arc(x,y,5,0,6.3);c.fill();c.strokeStyle=b.c;c.beginPath();c.arc(x,y,5+age*7,0,6.3);c.stroke();c.fillStyle='rgba('+I+',.9)';if(!narrow)c.fillText(b.n.toUpperCase()+(b.c==='#ff5d5d'?' · BLOCKED':''),x+10,y+4);c.globalAlpha=1});
 /* pings */ var now=performance.now();pings=pings.filter(function(p){return now-p.t<2200});pings.forEach(function(p){var u=(now-p.t)/2200;c.strokeStyle='rgba('+I+','+(.7*(1-u))+')';c.lineWidth=2;c.beginPath();c.arc(p.x,p.y,u*Math.min(300,w*.4),0,6.3);c.stroke();c.lineWidth=1;
  bl.forEach(function(b){var x=cx+Math.cos(b.a)*b.r*R,y=cy+Math.sin(b.a)*b.r*R;if(Math.abs(Math.hypot(x-p.x,y-p.y)-u*Math.min(300,w*.4))<10)b.seen=t})});
 /* centre */ c.fillStyle='rgba('+I+',.9)';c.beginPath();c.arc(cx,cy,3,0,6.3);c.fill();
 if(m.in){c.strokeStyle='rgba('+I+',.18)';c.beginPath();c.moveTo(m.x,m.y-12);c.lineTo(m.x,m.y+12);c.moveTo(m.x-12,m.y);c.lineTo(m.x+12,m.y);c.stroke()}
},{still:2.5,dpr:1.5,host:main});
})();
