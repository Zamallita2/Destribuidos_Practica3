import html2canvas from "html2canvas";
import { jsPDF } from "jspdf";
import { formatFlightLocalTime } from "@/lib/flightTime";

const PDF_STYLES = `
  #pdf-root{
    position:fixed;
    left:-20000px;
    top:0;
    width:1000px;
    padding:20px;
    background:#fff;
    pointer-events:none;
    z-index:-1;
    font-family:Arial,Helvetica,sans-serif;
  }

  .ticket-front,.ticket-back{
    width:880px;
    border-radius:15px;
    overflow:hidden;
    margin:0 auto;
    display:flex;
    position:relative;
    font-family:Arial,Helvetica,sans-serif;
  }

  .ticket-front{height:575px;margin-bottom:20px}
  .ticket-back{height:400px}

  .left,.back-left{
    flex:7;
    background:#f2f2f2;
    border-right:2px dashed #cfcfcf;
    position:relative;
    overflow:hidden;
  }

  .right,.back-right{
    flex:3;
    background:#16384b;
    color:#fff;
    overflow:hidden;
  }

  .left{padding:22px 26px 18px}
  .right{padding:22px 20px 18px}
  .back-right{padding:16px}

  .gold-top,.gold-bottom{
    position:absolute;
    left:0;
    width:100%;
    height:14px;
    background:#d4a62a;
  }

  .gold-top{top:0}
  .gold-bottom{bottom:0}

  .gold-lines-top,.gold-lines-bottom{
    position:absolute;
    width:90px;
    height:14px;
    background:repeating-linear-gradient(115deg,transparent 0 10px,#fff 10px 15px);
  }

  .gold-lines-top{top:0;left:58%}
  .gold-lines-bottom{bottom:0;left:60%}

  .wm{
    position:absolute;
    inset:0;
    display:flex;
    justify-content:center;
    align-items:center;
    font-size:220px;
    color:#17384b;
    opacity:.04;
    pointer-events:none;
  }

  .title-row{
    display:flex;
    justify-content:space-between;
    align-items:flex-start;
    margin:18px 0 16px;
    position:relative;
    z-index:2;
  }

  .title{
    font-size:34px;
    font-weight:900;
    color:#12141f;
    line-height:1;
    letter-spacing:1px;
  }

  .title span{color:#d4a62a}

  .logo{
    display:flex;
    align-items:center;
    gap:12px;
  }

  .logo-t{text-align:right}

  .logo-t h2{
    margin:0;
    font-size:15px;
    color:#111;
    font-weight:900;
  }

  .logo-t p{
    margin:2px 0 0;
    font-size:8px;
    letter-spacing:3px;
    color:#444;
  }

  .logo-i{
    width:36px;
    height:36px;
    border-radius:6px;
    background:#d4a62a;
    color:#fff;
    display:flex;
    align-items:center;
    justify-content:center;
    font-size:18px;
    font-weight:bold;
  }

  .main{
    display:grid;
    grid-template-columns:165px 1fr;
    gap:22px;
    position:relative;
    z-index:2;
  }

  .qr-side{
    display:flex;
    align-items:center;
    gap:14px;
  }

  .qr{
    width:120px;
    height:120px;
    background:#fff;
    padding:4px;
    box-shadow:0 0 0 1px #e7e7e7 inset;
    flex-shrink:0;
  }

  .qr img,.qr-sm img{
    width:100%;
    height:100%;
    display:block;
  }

  .scan-v{
    writing-mode:vertical-rl;
    text-orientation:mixed;
    font-size:16px;
    color:#111;
    letter-spacing:.5px;
    line-height:1;
    display:flex;
    align-items:center;
    justify-content:center;
    height:120px;
    white-space:nowrap;
  }

  .grid2{
    margin-top:18px;
    display:grid;
    grid-template-columns:1fr 1fr;
    gap:16px 14px;
  }

  .fields{
    display:flex;
    flex-direction:column;
    gap:16px;
  }

  .row3{
    display:grid;
    grid-template-columns:1fr .9fr 1.2fr;
    gap:16px;
  }

  .row2{
    display:grid;
    grid-template-columns:1.5fr .9fr;
    gap:16px;
    align-items:start;
  }

  .f{
    border-left:3px solid #d4a62a;
    padding-left:10px;
  }

  .f label,.sl{
    display:block;
    font-size:10px;
    color:#2a2a2a;
    margin-bottom:6px;
    text-transform:uppercase;
  }

  .f .v{
    font-size:18px;
    color:#111;
    line-height:1.05;
  }

  .f .m{
    font-size:14px;
    line-height:1.15;
  }

  .f .s{
    font-size:14px;
    line-height:1.15;
  }

  .groups{
    display:flex;
    gap:8px;
    flex-wrap:wrap;
  }

  .g{
    padding-bottom:10px;
    width:34px;
    height:34px;
    border:3px solid #1e2530;
    border-radius:50%;
    display:flex;
    align-items:center;
    justify-content:center;
    font-size:12px;
    font-weight:bold;
    color:#111;
  }

  .g.a{
    background:#d4a62a;
    border-color:#d4a62a;
    color:#fff;
  }

  .route-wrap{
    display:flex;
    justify-content:flex-end;
  }

  .route{
    width:380px;
    background:#16384b;
    color:#fff;
    display:grid;
    grid-template-columns:1fr 64px 1fr;
    align-items:center;
    padding:18px 20px;
    clip-path:polygon(7% 0,100% 0,100% 100%,7% 100%,0 50%);
    position:relative;
    min-height:110px;
  }

  .route:before{
    content:"";
    position:absolute;
    left:12px;
    top:0;
    width:7px;
    height:100%;
    background:#f2f2f2;
    clip-path:polygon(100% 0,0 50%,100% 100%,70% 100%,0 50%,70% 0);
  }

  .city{
    text-align:center;
    display:flex;
    flex-direction:column;
    justify-content:center;
    align-items:center;
  }

  .city h3{
    margin:0;
    margin-bottom:10px;
    font-size:50px;
    line-height:0.95;
    color:#d4a62a;
    font-weight:400;
  }

  .city p{
    margin:6px 0 0 0;
    font-size:11px;
    color:#fff;
    text-transform:uppercase;
    line-height:1.1;
  }

  .to{
    width:42px;
    height:42px;
    background:#d4a62a;
    border-radius:50%;
    display:flex;
    align-items:center;
    justify-content:center;
    margin:0 auto;
    color:#fff;
    font-size:16px;
    font-weight:bold;
    align-self:center;
  }
  .note{
    text-align:right;
    margin-top:8px;
    font-size:10px;
    color:#333;
    position:relative;
    z-index:2;
  }

  .st{
    font-size:28px;
    font-weight:900;
    line-height:1;
    margin-bottom:8px;
  }

  .sc{
    display:flex;
    align-items:center;
    gap:10px;
    margin-bottom:18px;
    font-size:15px;
  }

  .ss,.bs{
    background:repeating-linear-gradient(100deg,#d4a62a 0 7px,transparent 7px 11px);
  }

  .ss{
    width:70px;
    height:10px;
  }

  .sn{
    color:#d4a62a;
    font-size:16px;
    margin-bottom:14px;
    line-height:1.2;
  }

  .mr{
    background:#d4a62a;
    color:#fff;
    display:grid;
    grid-template-columns:1fr 42px 1fr;
    align-items:center;
    padding:10px 12px;
    clip-path:polygon(7% 0,100% 0,100% 100%,7% 100%,0 50%);
    position:relative;
    margin-bottom:16px;
  }

  .mr:before{
    content:"";
    position:absolute;
    left:8px;
    top:0;
    width:5px;
    height:100%;
    background:#16384b;
    clip-path:polygon(100% 0,0 50%,100% 100%,70% 100%,0 50%,70% 0);
  }

  .mc{text-align:center}

  .mc h4{
    margin:0;
    font-size:24px;
    font-weight:400;
  }

  .mc p{
    margin:2px 0 0;
    font-size:8px;
    text-transform:uppercase;
  }

  .mt{
    width:28px;
    height:28px;
    border:2px solid rgba(255,255,255,.75);
    border-radius:50%;
    display:flex;
    align-items:center;
    justify-content:center;
    margin:0 auto;
    font-size:10px;
  }

  .sg{
    display:grid;
    grid-template-columns:1fr 1fr;
    gap:12px 20px;
    margin-bottom:12px;
  }

  .sf .sl{
    font-size:10px;
    color:#d7e0e5;
  }

  .sf .v{
    font-size:16px;
    color:#d4a62a;
    line-height:1.1;
  }

  .bs{
    width:130px;
    height:10px;
    margin-bottom:12px;
  }

  .sb{
    display:flex;
    justify-content:space-between;
    align-items:flex-end;
    gap:12px;
  }

  .scan-s{
    font-size:11px;
    color:#d7e0e5;
    white-space:nowrap;
  }

  .qr-sm{
    width:78px;
    height:78px;
    background:#fff;
    padding:4px;
    flex-shrink:0;
  }

  .back-content{
    position:relative;
    z-index:2;
    padding:18px 24px 16px;
    height:100%;
  }

  .bt{
    font-size:24px;
    font-weight:900;
    margin:16px 0 14px;
    color:#111321;
  }

  .card{
    background:rgba(255,255,255,.68);
    border-left:6px solid #d4a62a;
    border-radius:12px;
    padding:14px 16px;
    margin-bottom:12px;
  }

  .card h3{
    margin:0 0 8px;
    font-size:14px;
    color:#1f465d;
    font-weight:800;
  }

  .card p,.card li{
    margin:0;
    font-size:10px;
    line-height:1.45;
    color:#444;
  }

  .card ul{
    margin:6px 0 0 16px;
    padding:0;
  }

  .bdt{
    font-size:22px;
    font-weight:900;
    margin:8px 0 12px;
  }

  .brc{
    display:flex;
    flex-direction:column;
    gap:10px;
  }

  .box{
    background:rgba(255,255,255,.08);
    border:1px solid rgba(255,255,255,.05);
    border-radius:12px;
    padding:12px;
  }

  .box h4{
    margin:0 0 8px;
    font-size:11px;
    color:#d4a62a;
    text-transform:uppercase;
  }

  .box p{
    margin:0;
    font-size:10px;
    line-height:1.45;
    color:#eef3f6;
  }

  .sum{
    display:grid;
    grid-template-columns:1fr 1fr;
    gap:10px;
  }

  .it{
    background:rgba(255,255,255,.08);
    border-radius:10px;
    padding:10px;
  }

  .it label{
    display:block;
    font-size:9px;
    color:#d7e0e5;
    margin-bottom:4px;
    text-transform:uppercase;
  }

  .it span{
    font-size:14px;
    color:#fff;
    font-weight:bold;
  }
`;

const esperarImagenes = async (root: HTMLElement) => {
  const imgs = Array.from(root.querySelectorAll("img"));
  await Promise.all(
    imgs.map((img) => {
      if (img.complete) return Promise.resolve();
      return new Promise<void>((resolve) => {
        img.onload = () => resolve();
        img.onerror = () => resolve();
      });
    })
  );
};

export async function downloadBoardingPassPdf(selectedBoleto: { id_boleto: number; estado: string }) {
  if (!selectedBoleto || selectedBoleto.estado !== "SALED") throw new Error("Solo los boletos comprados y vigentes tienen pase de abordar.");
  let div: HTMLDivElement | null = null;
  let styleTag: HTMLStyleElement | null = null;

  try {
    const passResponse = await fetch(`/api/boletos/${selectedBoleto.id_boleto}/pase`);
    if (!passResponse.ok) throw new Error("No se pudo generar el pase verificable");
    const pass = await passResponse.json();
    const escapeHtml = (value: unknown) => String(value).replace(/[&<>"']/g, (char) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" }[char] || char));
    const orgCod = escapeHtml(pass.origen);
    const dstCod = escapeHtml(pass.destino);
    const pasajero = escapeHtml(pass.pasajero);
    const asnt = escapeHtml(pass.asiento);
    const gate = escapeHtml(pass.puerta);
    const travelClass = pass.clase === "VIP" ? "FIRST CLASS" : "ECONOMY";
    const fl = `AP-${pass.vuelo}`;
    const date = formatFlightLocalTime(Math.floor(new Date(pass.salida_local).getTime() / 1000), pass.zona_salida);
    const arrivalDate = formatFlightLocalTime(Math.floor(new Date(pass.llegada_local).getTime() / 1000), pass.zona_llegada);
    const departureTime = date;
    const qrSource = `${pass.qr_url}`;

    styleTag = document.createElement("style");
    styleTag.id = "pdf-ticket-styles";
    styleTag.innerHTML = PDF_STYLES;
    document.head.appendChild(styleTag);

    div = document.createElement("div");
    div.id = "pdf-root";

    div.innerHTML = `
      <div class="ticket-front">
        <div class="left">
          <div class="gold-top"></div><div class="gold-bottom"></div><div class="gold-lines-top"></div><div class="gold-lines-bottom"></div><div class="wm">✈</div>
          <div class="title-row">
            <div class="title">BOARDING <span>PASS</span></div>
            <div class="logo">
              <div class="logo-t"><h2>AIRLINES PABON</h2><p>YOUR BEST CHOICE</p></div>
              <div class="logo-i">✈</div>
            </div>
          </div>
          <div class="main">
            <div>
              <div class="qr-side">
                <div class="qr">
                  <img src="${qrSource}" alt="QR">
                </div>
                <div class="scan-v">SCAN BARCODE</div>
              </div>
              <div class="grid2">
                <div class="f"><label>GATE :</label><div class="v">${gate}</div></div>
              <div class="f"><label>STATUS :</label><div class="v">PAGADO</div></div>
                <div class="f"><label>CLASS :</label><div class="v s">${travelClass}</div></div>
                <div class="f"><label>DEPARTURE :</label><div class="v">${departureTime}</div></div>
              </div>
            </div>
            <div class="fields">
              <div class="row3">
                <div class="f"><label>FLIGHT :</label><div class="v">${fl}</div></div>
                <div class="f"><label>SEAT :</label><div class="v">${asnt}</div></div>
                <div class="f"><label>DATE :</label><div class="v m">${date}</div></div>
              </div>
              <div class="row2">
                <div class="f"><label>PASSENGER NAME :</label><div class="v s">${pasajero}</div></div>
                <div class="f"><label>BOOKING :</label><div class="v">#${selectedBoleto.id_boleto}</div></div>
              </div>
              <div class="route-wrap">
                <div class="route">
                  <div class="city"><h3>${orgCod}</h3><p>${date}</p></div>
                  <div class="to">TO</div>
                  <div class="city"><h3>${dstCod}</h3><p>${arrivalDate}</p></div>
                </div>
              </div>
            </div>
          </div>
          <div class="note">Notes : Gate Closes 30 Minutes Before Departure</div>
        </div>

        <div class="right">
          <div class="st">BOARDING PASS</div>
          <div class="sc"><div class="ss"></div><div>${travelClass}</div></div>
      <div class="sl" style="color:#d7e0e5">PASSENGER NAME :</div>
          <div class="sn">${pasajero}</div>
          <div class="mr">
            <div class="mc"><h4>${orgCod}</h4><p>${date}</p></div>
            <div class="mt">TO</div>
            <div class="mc"><h4>${dstCod}</h4><p>${arrivalDate}</p></div>
          </div>
          <div class="sg">
            <div class="sf"><div class="sl">FLIGHT :</div><div class="v">${fl}</div></div>
            <div class="sf"><div class="sl">SEAT :</div><div class="v">${asnt}</div></div>
            <div class="sf"><div class="sl">DATE :</div><div class="v">${date}</div></div>
            <div class="sf"><div class="sl">GATE :</div><div class="v">${gate}</div></div>
            <div class="sf"><div class="sl">DEPARTURE :</div><div class="v">${departureTime}</div></div>
          </div>
          <div class="bs"></div>
          <div class="sb">
            <div class="scan-s">Scan Barcode Ticket</div>
            <div class="qr-sm">
              <img src="${qrSource}" alt="QR">
            </div>
          </div>
        </div>
      </div>

      <div class="ticket-back">
        <div class="back-left">
          <div class="gold-top"></div><div class="gold-bottom"></div><div class="gold-lines-top"></div><div class="gold-lines-bottom"></div><div class="wm">✈</div>
          <div class="back-content">
            <div class="bt">TICKET INFORMATION</div>
            <div class="card">
              <h3>Boarding Conditions</h3>
              <ul>
                <li>Arrive at the gate 30 minutes before departure.</li>
                <li>Passport or valid identification is required.</li>
                <li>Carry-on baggage is subject to security policies.</li>
              </ul>
            </div>
            <div class="card">
              <h3>Flight Summary</h3>
              <p>
                Passenger: ${pasajero}<br>
                Flight: ${fl}<br>
                Route: ${orgCod} to ${dstCod}<br>
                Seat: ${asnt}<br>
                Gate: ${gate}
              </p>
            </div>
          </div>
        </div>

        <div class="back-right">
          <div class="bdt">DETAILS</div>
          <div class="brc">
            <div class="box"><h4>PASSENGER</h4><p>${pasajero}</p></div>
            <div class="box"><h4>IMPORTANT NOTE</h4><p>This boarding pass is non-transferable and valid only for the named passenger.</p></div>
            <div class="sum">
              <div class="it"><label>CLASS</label><span>${travelClass}</span></div>
              <div class="it"><label>BOOKING</label><span>#${selectedBoleto.id_boleto}</span></div>
              <div class="it"><label>GATE</label><span>${gate}</span></div>
              <div class="it"><label>DEPARTURE</label><span>${departureTime}</span></div>
            </div>
          </div>
        </div>
      </div>
    `;

    document.body.appendChild(div);

    await esperarImagenes(div);

    const frente = div.querySelector(".ticket-front") as HTMLElement;
    const reverso = div.querySelector(".ticket-back") as HTMLElement;

    const pdf = new jsPDF("l", "mm", "a4");
    const pdfWidth = pdf.internal.pageSize.getWidth();
    const pdfHeight = pdf.internal.pageSize.getHeight();
    const margin = 6;

    const canvasFrente = await html2canvas(frente, {
      scale: 2,
      useCORS: true,
      backgroundColor: "#ffffff",
    });

    const imgFrente = canvasFrente.toDataURL("image/png");
    const frenteHeight = (canvasFrente.height * (pdfWidth - margin * 2)) / canvasFrente.width;

    pdf.addImage(
      imgFrente,
      "PNG",
      margin,
      margin,
      pdfWidth - margin * 2,
      Math.min(frenteHeight, pdfHeight - margin * 2)
    );

    pdf.addPage();

    const canvasReverso = await html2canvas(reverso, {
      scale: 2,
      useCORS: true,
      backgroundColor: "#ffffff",
    });

    const imgReverso = canvasReverso.toDataURL("image/png");
    const reversoHeight = (canvasReverso.height * (pdfWidth - margin * 2)) / canvasReverso.width;

    pdf.addImage(
      imgReverso,
      "PNG",
      margin,
      margin,
      pdfWidth - margin * 2,
      Math.min(reversoHeight, pdfHeight - margin * 2)
    );

    pdf.save(`Boleto_${selectedBoleto.id_boleto}.pdf`);
  } catch (err) {
    console.error(err);
    throw err;
  } finally {
    if (div) div.remove();
    const oldStyle = document.getElementById("pdf-ticket-styles");
    if (oldStyle) oldStyle.remove();
  }
}
