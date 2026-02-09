import { useEffect, useRef, useCallback } from 'react';
import maplibregl from 'maplibre-gl';

const HEXDB_API = 'https://hexdb.io/api/v1';
const PLANESPOTTERS_API = 'https://api.planespotters.net/pub/photos/hex';

function AircraftPopup({ map, aircraft, onClose, onPOV }) {
  const popupRef = useRef(null);
  const fetchController = useRef(null);
  const acRef = useRef(aircraft);
  const onCloseRef = useRef(onClose);
  const onPOVRef = useRef(onPOV);

  useEffect(() => { acRef.current = aircraft; }, [aircraft]);
  useEffect(() => { onCloseRef.current = onClose; }, [onClose]);
  useEffect(() => { onPOVRef.current = onPOV; }, [onPOV]);

  // Use event delegation — attach ONCE to popup element, survives setHTML replacements
  const setupPopupDelegation = useCallback((popup) => {
    const el = popup?.getElement();
    if (!el || el._povDelegated) return;
    el._povDelegated = true;
    el.addEventListener('click', (e) => {
      // Check if the click target is the POV button or inside it
      const btn = e.target.closest('#pov-btn');
      if (btn) {
        e.stopPropagation();
        e.preventDefault();
        if (onPOVRef.current && acRef.current) {
          onPOVRef.current(acRef.current);
        }
      }
    });
  }, []);

  const buildPopupHTML = useCallback((ac, info, photoUrl, photoCredit, routeStr, loading = false) => {
    const alt = ac.baroAltitude || ac.geoAltitude;
    const flStr = alt ? `FL${Math.round(alt * 3.28084 / 100)}` : '—';
    const spdKts = ac.velocity ? Math.round(ac.velocity * 1.944) : '—';
    const heading = ac.trueTrack != null ? `${Math.round(ac.trueTrack)}°` : '—';
    const vRate = ac.verticalRate != null 
      ? `${ac.verticalRate > 0 ? '+' : ''}${Math.round(ac.verticalRate * 196.85)} fpm` 
      : '—';
    const status = ac.onGround ? 'GND' : 'AIR';
    const statusColor = ac.onGround ? '#ff9500' : '#00d4ff';

    // Photo section: loading spinner → photo → styled placeholder
    let photoSection;
    if (loading) {
      photoSection = `<div class="ac-popup-photo ac-popup-photo-loading">
        <svg class="ac-popup-spinner" viewBox="0 0 50 50" width="40" height="40">
          <circle cx="25" cy="25" r="20" fill="none" stroke="rgba(0,212,255,0.15)" stroke-width="2"/>
          <circle cx="25" cy="25" r="20" fill="none" stroke="#00d4ff" stroke-width="2" stroke-dasharray="31 94" stroke-linecap="round" class="ac-popup-spinner-arc"/>
        </svg>
        <span class="ac-popup-loading-label">LOADING IMAGE</span>
      </div>`;
    } else if (photoUrl) {
      photoSection = `<div class="ac-popup-photo">
        <img src="${photoUrl}" alt="Aircraft" onerror="this.parentElement.classList.add('ac-popup-photo-fallback');this.style.display='none';this.parentElement.innerHTML='<svg viewBox=\\'0 0 24 24\\' width=\\'48\\' height=\\'48\\'><path d=\\'M12 2L4 12L6 14L10 12V20L8 21V22H16V21L14 20V12L18 14L20 12L12 2Z\\' fill=\\'rgba(0,212,255,0.3)\\'/></svg><span class=\\'ac-popup-loading-label\\'>${(ac.callsign || ac.icao24 || '').toUpperCase()}</span>';" />
        ${photoCredit ? `<span class="ac-popup-credit">📷 ${photoCredit}</span>` : ''}
      </div>`;
    } else {
      // No photo available — show aircraft icon placeholder
      photoSection = `<div class="ac-popup-photo ac-popup-photo-fallback">
        <svg viewBox="0 0 24 24" width="48" height="48">
          <path d="M12 2L4 12L6 14L10 12V20L8 21V22H16V21L14 20V12L18 14L20 12L12 2Z" fill="rgba(0,212,255,0.3)"/>
        </svg>
        <span class="ac-popup-loading-label">${(ac.callsign || ac.icao24 || '').toUpperCase()}</span>
      </div>`;
    }

    // Identity section
    let identitySection;
    if (loading) {
      identitySection = `<div class="ac-popup-identity-loading">
        <svg class="ac-popup-spinner-sm" viewBox="0 0 30 30" width="16" height="16">
          <circle cx="15" cy="15" r="11" fill="none" stroke="rgba(0,212,255,0.15)" stroke-width="2"/>
          <circle cx="15" cy="15" r="11" fill="none" stroke="#00d4ff" stroke-width="2" stroke-dasharray="17 52" stroke-linecap="round" class="ac-popup-spinner-arc"/>
        </svg>
        <span class="ac-popup-loading-text">Looking up aircraft details...</span>
      </div>`;
    } else if (info) {
      identitySection = `
        <div class="ac-popup-identity">
          <div><span class="ac-popup-sec">TYPE</span><br/><strong>${info.Type || info.ICAOTypeCode || '—'}</strong></div>
          <div><span class="ac-popup-sec">MFG</span><br/><strong>${info.Manufacturer || '—'}</strong></div>
        </div>
        ${info.RegisteredOwners ? `<div class="ac-popup-owner"><span class="ac-popup-sec">OPR</span> ${info.RegisteredOwners}</div>` : ''}
      `;
    } else {
      identitySection = '';
    }

    const routeSection = routeStr ? `<div class="ac-popup-route">${routeStr}</div>` : '';

    return `
      <div class="ac-popup-card">
        ${photoSection}
        <div class="ac-popup-header">
          <div>
            <div class="ac-popup-callsign">${ac.callsign || 'UNKNOWN'}</div>
            <div class="ac-popup-icao">${ac.icao24?.toUpperCase()}${info?.Registration ? ` · ${info.Registration}` : ''}</div>
          </div>
          <span class="ac-popup-status" style="color:${statusColor};border-color:${statusColor}">${status}</span>
        </div>
        ${identitySection}
        ${routeSection}
        <div class="ac-popup-data">
          <div><span class="ac-popup-val">${flStr}</span><span class="ac-popup-lbl">FL</span></div>
          <div><span class="ac-popup-val">${spdKts}</span><span class="ac-popup-lbl">KTS</span></div>
          <div><span class="ac-popup-val">${heading}</span><span class="ac-popup-lbl">HDG</span></div>
          <div><span class="ac-popup-val">${vRate}</span><span class="ac-popup-lbl">V/S</span></div>
        </div>
        <div class="ac-popup-footer">
          <span>${ac.originCountry || 'Unknown'}</span>
          <button class="ac-popup-pov-btn" id="pov-btn">◉ PILOT POV</button>
          <span class="ac-popup-sec">${ac.squawk ? `SQ ${ac.squawk}` : ''}</span>
        </div>
      </div>
    `;
  }, []);

  useEffect(() => {
    if (!map || !aircraft) {
      if (popupRef.current) {
        popupRef.current.remove();
        popupRef.current = null;
      }
      return;
    }

    if (fetchController.current) fetchController.current.abort();
    fetchController.current = new AbortController();
    const signal = fetchController.current.signal;

    const lon = aircraft.longitude;
    const lat = aircraft.latitude;
    if (lon == null || lat == null) return;

    if (popupRef.current) {
      popupRef.current.remove();
      popupRef.current = null;
    }

    const popup = new maplibregl.Popup({
      closeButton: true,
      closeOnClick: false,
      closeOnMove: false,
      maxWidth: '340px',
      anchor: 'left',
      offset: [15, 0],
      className: 'ac-popup-container',
    })
    .setLngLat([lon, lat])
    .setHTML(buildPopupHTML(aircraft, null, null, null, null, true))
    .addTo(map);

    popup.on('close', () => {
      popupRef.current = null;
      onCloseRef.current();
    });

    popupRef.current = popup;
    setTimeout(() => setupPopupDelegation(popup), 50);

    // Enrich with API data
    const hex = aircraft.icao24;
    const callsign = aircraft.callsign?.trim();

    Promise.all([
      fetch(`${HEXDB_API}/aircraft/${hex}`, { signal }).then(r => r.ok ? r.json() : null).catch(() => null),
      callsign 
        ? fetch(`${HEXDB_API}/route/icao/${callsign}`, { signal }).then(r => r.ok ? r.json() : null).catch(() => null)
        : Promise.resolve(null),
      fetch(`${PLANESPOTTERS_API}/${hex}`, { signal }).then(r => r.ok ? r.json() : null).catch(() => null),
    ]).then(async ([acInfo, routeData, photoData]) => {
      if (signal.aborted || !popupRef.current) return;

      const info = acInfo && !acInfo.status ? acInfo : null;

      // Photo — only use planespotters (hexdb has CORS issues from browser)
      let photoUrl = null, photoCredit = null;
      if (photoData?.photos?.length > 0) {
        const photo = photoData.photos[0];
        photoUrl = photo.thumbnail_large?.src || photo.thumbnail?.src || null;
        photoCredit = photo.photographer || null;
      }
      // photoUrl stays null if no planespotters photo → shows airplane icon placeholder

      // Route
      let routeStr = null;
      if (routeData && !routeData.status && routeData.route) {
        const parts = routeData.route.split('-');
        const dep = parts[0]?.trim();
        const arr = parts[parts.length - 1]?.trim();
        if (dep && arr) {
          // Resolve airport names in parallel
          const [depApt, arrApt] = await Promise.all([
            fetch(`${HEXDB_API}/airport/icao/${dep}`, { signal }).then(r => r.ok ? r.json() : null).catch(() => null),
            fetch(`${HEXDB_API}/airport/icao/${arr}`, { signal }).then(r => r.ok ? r.json() : null).catch(() => null),
          ]);
          const depName = depApt?.airport || dep;
          const arrName = arrApt?.airport || arr;
          routeStr = `<div class="ac-popup-route-codes">${dep}  ✈  ${arr}</div>`;
          routeStr += `<div class="ac-popup-route-names">${depName}  →  ${arrName}</div>`;
        }
      }

      if (popupRef.current) {
        popupRef.current.setHTML(buildPopupHTML(aircraft, info, photoUrl, photoCredit, routeStr));
        setTimeout(() => setupPopupDelegation(popupRef.current), 50);
      }
    });

    return () => {
      if (fetchController.current) fetchController.current.abort();
    };
  }, [aircraft?.icao24, map, buildPopupHTML, setupPopupDelegation]);

  // Update popup position as aircraft moves
  useEffect(() => {
    if (popupRef.current && aircraft?.longitude != null && aircraft?.latitude != null) {
      popupRef.current.setLngLat([aircraft.longitude, aircraft.latitude]);
    }
  }, [aircraft?.longitude, aircraft?.latitude]);

  useEffect(() => {
    return () => {
      if (popupRef.current) {
        popupRef.current.remove();
        popupRef.current = null;
      }
    };
  }, []);

  return null;
}

export default AircraftPopup;
