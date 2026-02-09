import React, { useRef, useMemo, useEffect, useState, useCallback } from 'react';
import { Canvas, useFrame, useThree } from '@react-three/fiber';
import { OrbitControls, Stars, Html } from '@react-three/drei';
import * as THREE from 'three';

const EARTH_TEXTURE = 'https://unpkg.com/three-globe@2.31.0/example/img/earth-blue-marble.jpg';
const EARTH_BUMP = 'https://unpkg.com/three-globe@2.31.0/example/img/earth-topology.png';

const GLOBE_RADIUS = 2;
const AC_ALT = 0.03; // altitude above globe surface

function latLonToVector3(lat, lon, radius) {
  const phi = (90 - lat) * (Math.PI / 180);
  const theta = (lon + 180) * (Math.PI / 180);
  return new THREE.Vector3(
    -radius * Math.sin(phi) * Math.cos(theta),
    radius * Math.cos(phi),
    radius * Math.sin(phi) * Math.sin(theta)
  );
}

// Create an airplane shape (triangle arrow)
function createAircraftShape() {
  const shape = new THREE.Shape();
  // Simple arrow/jet shape pointing up (+Y)
  shape.moveTo(0, 0.014);        // nose
  shape.lineTo(-0.004, 0.004);   // left wing leading
  shape.lineTo(-0.01, 0.002);    // left wingtip
  shape.lineTo(-0.003, 0);       // left wing trailing
  shape.lineTo(-0.003, -0.008);  // left tail
  shape.lineTo(-0.006, -0.012);  // left stabilizer tip
  shape.lineTo(-0.002, -0.009);  // inner tail left
  shape.lineTo(0, -0.01);        // tail center
  shape.lineTo(0.002, -0.009);   // inner tail right
  shape.lineTo(0.006, -0.012);   // right stabilizer tip
  shape.lineTo(0.003, -0.008);   // right tail
  shape.lineTo(0.003, 0);        // right wing trailing
  shape.lineTo(0.01, 0.002);     // right wingtip
  shape.lineTo(0.004, 0.004);    // right wing leading
  shape.closePath();
  return shape;
}

// Shared geometry + materials
const aircraftShape = createAircraftShape();
const aircraftGeometry = new THREE.ShapeGeometry(aircraftShape);
const matCyan = new THREE.MeshBasicMaterial({ color: '#00d4ff', side: THREE.DoubleSide });
const matGreen = new THREE.MeshBasicMaterial({ color: '#00ff88', side: THREE.DoubleSide });
const matOrange = new THREE.MeshBasicMaterial({ color: '#ff9500', side: THREE.DoubleSide });

// Earth Globe
function Earth({ earthRef }) {
  const meshRef = useRef();
  const [texture, setTexture] = useState(null);
  const [bumpMap, setBumpMap] = useState(null);

  useEffect(() => {
    const loader = new THREE.TextureLoader();
    loader.load(EARTH_TEXTURE, (tex) => {
      tex.colorSpace = THREE.SRGBColorSpace;
      setTexture(tex);
    });
    loader.load(EARTH_BUMP, (tex) => setBumpMap(tex));
  }, []);

  const fallbackTexture = useMemo(() => {
    const canvas = document.createElement('canvas');
    canvas.width = 1024;
    canvas.height = 512;
    const ctx = canvas.getContext('2d');
    const gradient = ctx.createLinearGradient(0, 0, 0, canvas.height);
    gradient.addColorStop(0, '#0a1628');
    gradient.addColorStop(0.5, '#0f2540');
    gradient.addColorStop(1, '#0a1628');
    ctx.fillStyle = gradient;
    ctx.fillRect(0, 0, canvas.width, canvas.height);
    ctx.strokeStyle = 'rgba(0, 212, 255, 0.15)';
    ctx.lineWidth = 1;
    for (let i = 0; i <= 18; i++) {
      ctx.beginPath(); ctx.moveTo(0, (i/18)*512); ctx.lineTo(1024, (i/18)*512); ctx.stroke();
    }
    for (let i = 0; i <= 36; i++) {
      ctx.beginPath(); ctx.moveTo((i/36)*1024, 0); ctx.lineTo((i/36)*1024, 512); ctx.stroke();
    }
    const tex = new THREE.CanvasTexture(canvas);
    tex.needsUpdate = true;
    return tex;
  }, []);

  useFrame(() => {
    if (meshRef.current) {
      meshRef.current.rotation.y += 0.0003;
      if (earthRef) earthRef.current = meshRef.current;
    }
  });

  return (
    <group>
      <mesh ref={meshRef}>
        <sphereGeometry args={[GLOBE_RADIUS, 64, 64]} />
        <meshStandardMaterial
          map={texture || fallbackTexture}
          bumpMap={bumpMap}
          bumpScale={0.05}
          roughness={0.7}
          metalness={0.1}
        />
      </mesh>
      <mesh scale={1.015}>
        <sphereGeometry args={[GLOBE_RADIUS, 32, 32]} />
        <meshBasicMaterial color="#4da6ff" transparent opacity={0.08} side={THREE.BackSide} />
      </mesh>
      <mesh scale={1.04}>
        <sphereGeometry args={[GLOBE_RADIUS, 32, 32]} />
        <meshBasicMaterial color="#00d4ff" transparent opacity={0.03} side={THREE.BackSide} />
      </mesh>
    </group>
  );
}

// Single aircraft icon on globe surface
function AircraftIcon({ aircraft, isSelected, onClick, earthRef }) {
  const groupRef = useRef();

  const position = useMemo(() => {
    if (aircraft.latitude == null || aircraft.longitude == null) return null;
    return latLonToVector3(aircraft.latitude, aircraft.longitude, GLOBE_RADIUS + AC_ALT);
  }, [aircraft.latitude, aircraft.longitude]);

  // Normal vector (pointing outward from globe center)
  const normal = useMemo(() => {
    if (!position) return null;
    return position.clone().normalize();
  }, [position]);

  const material = isSelected ? matGreen : (aircraft.onGround ? matOrange : matCyan);

  useFrame(() => {
    if (!groupRef.current || !position || !normal || !earthRef?.current) return;

    // Get the earth's current rotation
    const earthRotY = earthRef.current.rotation.y;

    // Rotate position around Y axis to match earth rotation
    const rotatedPos = position.clone().applyAxisAngle(new THREE.Vector3(0, 1, 0), earthRotY);
    groupRef.current.position.copy(rotatedPos);

    // Orient the aircraft to sit tangent to the globe surface
    const rotatedNormal = normal.clone().applyAxisAngle(new THREE.Vector3(0, 1, 0), earthRotY);

    // Look outward from globe center, then tilt to lie flat on surface
    const up = new THREE.Vector3(0, 1, 0);
    const quat = new THREE.Quaternion();
    quat.setFromUnitVectors(new THREE.Vector3(0, 0, 1), rotatedNormal);
    groupRef.current.quaternion.copy(quat);

    // Apply aircraft heading rotation around the normal axis
    const headingRad = ((aircraft.trueTrack || 0) * Math.PI) / 180;
    const headingQuat = new THREE.Quaternion();
    headingQuat.setFromAxisAngle(rotatedNormal, -headingRad);
    groupRef.current.quaternion.premultiply(headingQuat);
  });

  if (!position) return null;

  return (
    <group ref={groupRef}>
      <mesh
        geometry={aircraftGeometry}
        material={material}
        onClick={(e) => {
          e.stopPropagation();
          onClick(aircraft);
        }}
      />
      {isSelected && (
        <Html position={[0, 0.03, 0]} center style={{ pointerEvents: 'none', whiteSpace: 'nowrap' }}>
          <div style={{
            background: 'rgba(0, 0, 0, 0.85)',
            border: '1px solid #00d4ff',
            borderRadius: '4px',
            padding: '3px 8px',
            color: '#00ff88',
            fontFamily: 'Orbitron, monospace',
            fontSize: '10px',
            fontWeight: 600,
            letterSpacing: '1px',
          }}>
            {aircraft.callsign || aircraft.icao24}
          </div>
        </Html>
      )}
    </group>
  );
}

// Camera Controller
function CameraController({ region }) {
  const { camera } = useThree();

  useEffect(() => {
    if (!region?.center) return;
    const [lon, lat] = region.center;
    const phi = (90 - lat) * (Math.PI / 180);
    const theta = (lon + 180) * (Math.PI / 180);
    const dist = 4.5;
    const endX = -dist * Math.sin(phi) * Math.cos(theta);
    const endY = dist * Math.cos(phi);
    const endZ = dist * Math.sin(phi) * Math.sin(theta);

    const start = { x: camera.position.x, y: camera.position.y, z: camera.position.z };
    const duration = 1500;
    const startTime = Date.now();

    function animate() {
      const t = Math.min((Date.now() - startTime) / duration, 1);
      const ease = t < 0.5 ? 2 * t * t : 1 - Math.pow(-2 * t + 2, 2) / 2;
      camera.position.set(
        start.x + (endX - start.x) * ease,
        start.y + (endY - start.y) * ease,
        start.z + (endZ - start.z) * ease,
      );
      camera.lookAt(0, 0, 0);
      if (t < 1) requestAnimationFrame(animate);
    }
    animate();
  }, [region?.center?.[0], region?.center?.[1], camera]);

  return null;
}

// Main Globe3D
function Globe3D({ aircraft, region, selectedAircraft, onSelectAircraft }) {
  const earthRef = useRef(null);

  return (
    <div className="c2-globe-container">
      <Canvas
        camera={{ fov: 45, near: 0.1, far: 100 }}
        gl={{ antialias: true, alpha: true }}
      >
        <ambientLight intensity={0.3} />
        <directionalLight position={[5, 3, 5]} intensity={0.8} />
        <pointLight position={[-5, -3, -5]} intensity={0.4} color="#00d4ff" />

        <Stars radius={50} depth={50} count={2000} factor={4} saturation={0} fade />

        <Earth earthRef={earthRef} />

        {aircraft.map((ac) => (
          <AircraftIcon
            key={ac.icao24}
            aircraft={ac}
            isSelected={selectedAircraft?.icao24 === ac.icao24}
            onClick={onSelectAircraft}
            earthRef={earthRef}
          />
        ))}

        <OrbitControls
          enableZoom
          enablePan={false}
          minDistance={3}
          maxDistance={10}
          autoRotate={false}
        />

        <CameraController region={region} />
      </Canvas>

      <div style={{
        position: 'absolute',
        bottom: '24px',
        right: '24px',
        padding: '8px 12px',
        background: 'rgba(0, 0, 0, 0.7)',
        border: '1px solid rgba(0, 212, 255, 0.3)',
        borderRadius: '4px',
        color: '#00d4ff',
        fontFamily: 'JetBrains Mono, monospace',
        fontSize: '10px',
        letterSpacing: '1px',
      }}>
        WebGL 3D VIEW
      </div>
    </div>
  );
}

export default Globe3D;
