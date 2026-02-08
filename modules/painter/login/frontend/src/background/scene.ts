import * as THREE from 'three';

// 定义颜色接口
interface Colors {
    red: number;
    yellow: number;
    white: number;
    brown: number;
    pink: number;
    brownDark: number;
    blue: number;
    green: number;
    purple: number;
    lightgreen: number;
}

// 定义颜色常量
const Colors: Colors = {
    red: 0xf25346,
    yellow: 0xedeb27,
    white: 0xd8d0d1,
    brown: 0x59332e,
    pink: 0xF5986E,
    brownDark: 0x23190f,
    blue: 0x68c3c0,
    green: 0x458248,
    purple: 0x551A8B,
    lightgreen: 0x629265,
};

// 定义场景、相机和渲染器的返回类型
interface SceneObjects {
    scene: THREE.Scene;
    camera: THREE.PerspectiveCamera;
    renderer: THREE.WebGLRenderer;
}

// 创建场景
function createScene(ref: HTMLDivElement): SceneObjects {
    const HEIGHT = window.innerHeight;
    const WIDTH = window.innerWidth;

    const scene = new THREE.Scene();
    scene.fog = new THREE.Fog(0xf7d9aa, 100, 950);

    const aspectRatio = WIDTH / HEIGHT;
    const fieldOfView = 60;
    const nearPlane = 1;
    const farPlane = 10000;
    const camera = new THREE.PerspectiveCamera(fieldOfView, aspectRatio, nearPlane, farPlane);
    camera.position.set(0, 150, 100);

    const renderer = new THREE.WebGLRenderer({
        alpha: true,
        antialias: true
    });

    renderer.setSize(WIDTH, HEIGHT);
    renderer.shadowMap.enabled = true;

    const container = ref
    // const container = document.getElementById(styles.background);
    // if (!container) {
    //     throw new Error('Element with id "world" not found');
    // }

    renderer.domElement.style.background = "linear-gradient(#e4e0ba, #f7d9aa)";
    renderer.domElement.style.position = "fixed";
    renderer.domElement.style.top = "0";
    renderer.domElement.style.left = "0";
    renderer.domElement.style.zIndex = "-2";
    container.appendChild(renderer.domElement);

    const handleWindowResize = () => {
        const HEIGHT = window.innerHeight;
        const WIDTH = window.innerWidth;
        renderer.setSize(WIDTH, HEIGHT);
        camera.aspect = WIDTH / HEIGHT;
        camera.updateProjectionMatrix();
    };

    window.addEventListener('resize', handleWindowResize);
    screen.orientation?.addEventListener('change', handleWindowResize);

    return { scene, camera, renderer };
}

// 创建灯光
function createLights(scene: THREE.Scene) {
    const hemisphereLight = new THREE.HemisphereLight(0xaaaaaa, 0x000000, .9);
    const shadowLight = new THREE.DirectionalLight(0xffffff, .9);

    shadowLight.position.set(0, 350, 350);
    shadowLight.castShadow = true;

    shadowLight.shadow.camera.left = -650;
    shadowLight.shadow.camera.right = 650;
    shadowLight.shadow.camera.top = 650;
    shadowLight.shadow.camera.bottom = -650;
    shadowLight.shadow.camera.near = 1;
    shadowLight.shadow.camera.far = 1000;

    shadowLight.shadow.mapSize.width = 2048;
    shadowLight.shadow.mapSize.height = 2048;

    scene.add(hemisphereLight);
    scene.add(shadowLight);
}

// Land 类
class Land {
    mesh: THREE.Mesh;

    constructor() {
        const geom = new THREE.CylinderGeometry(600, 600, 1700, 40, 10);
        geom.applyMatrix4(new THREE.Matrix4().makeRotationX(-Math.PI / 2));

        const mat = new THREE.MeshPhongMaterial({
            color: Colors.lightgreen,
            flatShading: true,
        });

        this.mesh = new THREE.Mesh(geom, mat);
        this.mesh.receiveShadow = true;
    }
}

// Orbit 类
class Orbit {
    mesh: THREE.Object3D;

    constructor() {
        this.mesh = new THREE.Object3D();
    }
}

// Sun 类
class Sun {
    mesh: THREE.Object3D;

    constructor() {
        this.mesh = new THREE.Object3D();

        const sunGeom = new THREE.SphereGeometry(400, 20, 10);
        const sunMat = new THREE.MeshPhongMaterial({
            color: Colors.yellow,
            flatShading: true,
        });
        const sun = new THREE.Mesh(sunGeom, sunMat);
        sun.castShadow = false;
        sun.receiveShadow = false;
        this.mesh.add(sun);
    }
}

// Cloud 类
class Cloud {
    mesh: THREE.Object3D;

    constructor() {
        this.mesh = new THREE.Object3D();
        const geom = new THREE.DodecahedronGeometry(20, 0);
        const mat = new THREE.MeshPhongMaterial({
            color: Colors.white,
            flatShading: true,
        });

        const nBlocs = 3 + Math.floor(Math.random() * 3);

        for (let i = 0; i < nBlocs; i++) {
            const m = new THREE.Mesh(geom, mat);
            m.position.set(i * 15, Math.random() * 10, Math.random() * 10);
            m.rotation.z = Math.random() * Math.PI * 2;
            m.rotation.y = Math.random() * Math.PI * 2;

            const s = .1 + Math.random() * .9;
            m.scale.set(s, s, s);
            this.mesh.add(m);
        }
    }
}

// Sky 类
class Sky {
    mesh: THREE.Object3D;
    nClouds: number; // 显式声明属性

    constructor() {
        this.mesh = new THREE.Object3D();
        this.nClouds = 25; // 初始化

        const stepAngle = Math.PI * 2 / this.nClouds;

        for (let i = 0; i < this.nClouds; i++) {
            const c = new Cloud();
            const a = stepAngle * i;
            const h = 800 + Math.random() * 200;

            c.mesh.position.set(
                Math.cos(a) * h,
                Math.sin(a) * h,
                -400 - Math.random() * 400
            );
            c.mesh.rotation.z = a + Math.PI / 2;

            const s = 1 + Math.random() * 2;
            c.mesh.scale.set(s, s, s);

            this.mesh.add(c.mesh);
        }
    }
}

// Tree 类
class Tree {
    mesh: THREE.Object3D;

    constructor() {
        this.mesh = new THREE.Object3D();

        const matTreeLeaves = new THREE.MeshPhongMaterial({
            color: Colors.green,
            flatShading: true
        });

        const geonTreeBase = new THREE.BoxGeometry(10, 20, 10);
        const matTreeBase = new THREE.MeshBasicMaterial({ color: Colors.brown });
        const treeBase = new THREE.Mesh(geonTreeBase, matTreeBase);
        treeBase.castShadow = true;
        treeBase.receiveShadow = true;
        this.mesh.add(treeBase);

        const geomTreeLeaves1 = new THREE.CylinderGeometry(1, 12 * 3, 12 * 3, 4);
        const treeLeaves1 = new THREE.Mesh(geomTreeLeaves1, matTreeLeaves);
        treeLeaves1.castShadow = true;
        treeLeaves1.receiveShadow = true;
        treeLeaves1.position.y = 20;
        this.mesh.add(treeLeaves1);

        const geomTreeLeaves2 = new THREE.CylinderGeometry(1, 9 * 3, 9 * 3, 4);
        const treeLeaves2 = new THREE.Mesh(geomTreeLeaves2, matTreeLeaves);
        treeLeaves2.castShadow = true;
        treeLeaves2.position.y = 40;
        treeLeaves2.receiveShadow = true;
        this.mesh.add(treeLeaves2);

        const geomTreeLeaves3 = new THREE.CylinderGeometry(1, 6 * 3, 6 * 3, 4);
        const treeLeaves3 = new THREE.Mesh(geomTreeLeaves3, matTreeLeaves);
        treeLeaves3.castShadow = true;
        treeLeaves3.position.y = 55;
        treeLeaves3.receiveShadow = true;
        this.mesh.add(treeLeaves3);
    }
}

// Flower 类
class Flower {
    mesh: THREE.Object3D;
    constructor() {
        this.mesh = new THREE.Object3D();
        //花茎 一个绿色立方体（BoxGeometry(5, 50, 5)），高度 50，作为花的支撑
        const geomStem = new THREE.BoxGeometry(5, 50, 5);
        const matStem = new THREE.MeshPhongMaterial({
            color: Colors.green,
            flatShading: true
        });
        const stem = new THREE.Mesh(geomStem, matStem);
        stem.castShadow = false; //不投射阴影
        stem.receiveShadow = true; //接受阴影
        this.mesh.add(stem);

        //花心（PetalCore） 一个黄色小盒子（BoxGeometry(5, 5, 10)），作为花瓣的基座
        //位置在花茎顶部（position.set(0, 25, 3)），y = 25 对应花茎高度的一半
        const geomPetalCore = new THREE.BoxGeometry(6, 6, 10);
        const matPetalCore = new THREE.MeshPhongMaterial({
            color: Colors.yellow,
            flatShading: true
        });
        const petalCore = new THREE.Mesh(geomPetalCore, matPetalCore);
        petalCore.castShadow = false;
        petalCore.receiveShadow = true;

        //花瓣（Petals）：
        const petalColors: number[] = [Colors.red, Colors.yellow, Colors.blue];
        const petalColor = petalColors[Math.floor(Math.random() * 3)];

        const geomPetal = new THREE.BoxGeometry(10, 30, 5);
        const matPetal = new THREE.MeshBasicMaterial({ color: petalColor });
        const positions = geomPetal.attributes.position as THREE.BufferAttribute;
        positions.setUsage(THREE.DynamicDrawUsage);

        const petals = [];
        for (let i = 0; i < 2; i++) {
            const petal = new THREE.Mesh(geomPetal, matPetal);
            petal.rotation.z = i * Math.PI / 2;
            petal.castShadow = true; //投射阴影
            petal.receiveShadow = true; //接受阴影
            petals.push(petal);
        }

        petalCore.add(...petals);
        petalCore.position.set(0, 25, 3);
        this.mesh.add(petalCore);
    }
}

// Forest 类
class Forest {
    mesh: THREE.Object3D;
    nTrees: number; // 显式声明属性
    nFlowers: number; // 显式声明属性

    constructor() {
        this.mesh = new THREE.Object3D();
        this.nTrees = 300;
        this.nFlowers = 350;

        const stepAngle = Math.PI * 2 / this.nTrees;

        for (let i = 0; i < this.nTrees; i++) {
            const t = new Tree();
            const a = stepAngle * i;
            const h = 605;

            t.mesh.position.set(
                Math.cos(a) * h,
                Math.sin(a) * h,
                0 - Math.random() * 600
            );
            t.mesh.rotation.z = a + (Math.PI / 2) * 3;

            const s = .3 + Math.random() * .75;
            t.mesh.scale.set(s, s, s);

            this.mesh.add(t.mesh);
        }

        const stepAngleFlowers = Math.PI * 2 / this.nFlowers;

        for (let i = 0; i < this.nFlowers; i++) {
            const f = new Flower();
            const a = stepAngleFlowers * i;
            const h = 605;

            f.mesh.position.set(
                Math.cos(a) * h,
                Math.sin(a) * h,
                0 - Math.random() * 600
            );
            f.mesh.rotation.z = a + (Math.PI / 2) * 3;

            const s = .1 + Math.random() * .3;
            f.mesh.scale.set(s, s, s);

            this.mesh.add(f.mesh);
        }
    }
}

// AirPlane 类
class AirPlane {
    mesh: THREE.Object3D;
    propeller: THREE.Mesh;
    targetX: number;
    targetY: number;
    t: number;
    vx: number;
    vy: number;
    ax: number;
    ay: number;

    constructor() {
        this.mesh = new THREE.Object3D();

        // Create the cabin
        const geomCockpit = new THREE.BoxGeometry(80, 50, 50);
        const matCockpit = new THREE.MeshPhongMaterial({
            color: Colors.red,
            flatShading: true
        });

        const positions = geomCockpit.attributes.position as THREE.BufferAttribute;
        positions.setUsage(THREE.DynamicDrawUsage);
        // array[4 * 3 + 1] -= 10; // 顶点4 y
        // array[4 * 3 + 2] += 20; // 顶点4 z
        // array[5 * 3 + 1] -= 10; // 顶点5 y
        // array[5 * 3 + 2] -= 20; // 顶点5 z
        // array[6 * 3 + 1] += 30; // 顶点6 y
        // array[6 * 3 + 2] += 20; // 顶点6 z
        // array[7 * 3 + 1] += 30; // 顶点7 y
        // array[7 * 3 + 2] -= 20; // 顶点7 z
        // 修改顶点4
        positions.setXYZ(4,
            positions.getX(4),
            positions.getY(4) - 10,  // y 坐标减10
            positions.getZ(4) + 20   // z 坐标加20
        );

        // 修改顶点5
        positions.setXYZ(5,
            positions.getX(5),
            positions.getY(5) - 10,  // y 坐标减10
            positions.getZ(5) - 20   // z 坐标减20
        );

        // 修改顶点6
        positions.setXYZ(
            6,
            positions.getX(6),
            positions.getY(6) + 30,  // y 坐标加30
            positions.getZ(6) + 20   // z 坐标加20
        );

        // 修改顶点7
        positions.setXYZ(
            7,
            positions.getX(7),
            positions.getY(7) + 30,  // y 坐标加30
            positions.getZ(7) - 20   // z 坐标减20
        );

        positions.needsUpdate = true;

        const cockpit = new THREE.Mesh(geomCockpit, matCockpit);
        cockpit.castShadow = true;
        cockpit.receiveShadow = true;
        this.mesh.add(cockpit);

        // Create the engine
        const geomEngine = new THREE.BoxGeometry(20, 50, 50);
        const matEngine = new THREE.MeshPhongMaterial({
            color: Colors.white,
            flatShading: true
        });
        const engine = new THREE.Mesh(geomEngine, matEngine);
        engine.position.x = 40;
        engine.castShadow = true;
        engine.receiveShadow = true;
        this.mesh.add(engine);

        // Create the tail
        const geomTailPlane = new THREE.BoxGeometry(15, 20, 5);
        const matTailPlane = new THREE.MeshPhongMaterial({
            color: Colors.red,
            flatShading: true
        });
        const tailPlane = new THREE.Mesh(geomTailPlane, matTailPlane);
        tailPlane.position.set(-35, 25, 0);
        tailPlane.castShadow = true;
        tailPlane.receiveShadow = true;
        this.mesh.add(tailPlane);

        // Create the wing
        const geomSideWing = new THREE.BoxGeometry(40, 4, 150);
        const matSideWing = new THREE.MeshPhongMaterial({
            color: Colors.red,
            flatShading: true
        });

        const sideWingTop = new THREE.Mesh(geomSideWing, matSideWing);
        const sideWingBottom = new THREE.Mesh(geomSideWing, matSideWing);
        sideWingTop.castShadow = true;
        sideWingTop.receiveShadow = true;
        sideWingBottom.castShadow = true;
        sideWingBottom.receiveShadow = true;

        sideWingTop.position.set(20, 12, 0);
        sideWingBottom.position.set(20, -3, 0);
        this.mesh.add(sideWingTop);
        this.mesh.add(sideWingBottom);

        const geomWindshield = new THREE.BoxGeometry(3, 15, 20);
        const matWindshield = new THREE.MeshPhongMaterial({
            color: Colors.white,
            transparent: true,
            opacity: .3,
            flatShading: true
        });
        const windshield = new THREE.Mesh(geomWindshield, matWindshield);
        windshield.position.set(5, 27, 0);
        windshield.castShadow = true;
        windshield.receiveShadow = true;
        this.mesh.add(windshield);

        const geomPropeller = new THREE.BoxGeometry(20, 10, 10);
        const matPropeller = new THREE.MeshPhongMaterial({
            color: Colors.brown,
            flatShading: true
        });

        // 修复：移除vertices修改，改用BufferGeometry的attributes
        const propellerPositions = geomPropeller.attributes.position;
        const propellerArray = propellerPositions.array as Float32Array;

        // 修改顶点坐标
        propellerArray[4 * 3 + 1] -= 5; // 顶点4 y
        propellerArray[4 * 3 + 2] += 5; // 顶点4 z
        propellerArray[5 * 3 + 1] -= 5; // 顶点5 y
        propellerArray[5 * 3 + 2] -= 5; // 顶点5 z
        propellerArray[6 * 3 + 1] += 5; // 顶点6 y
        propellerArray[6 * 3 + 2] += 5; // 顶点6 z
        propellerArray[7 * 3 + 1] += 5; // 顶点7 y
        propellerArray[7 * 3 + 2] -= 5; // 顶点7 z

        propellerPositions.needsUpdate = true;

        this.propeller = new THREE.Mesh(geomPropeller, matPropeller);
        this.propeller.castShadow = true;
        this.propeller.receiveShadow = true;

        const geomBlade1 = new THREE.BoxGeometry(1, 100, 10);
        const geomBlade2 = new THREE.BoxGeometry(1, 10, 100);
        const matBlade = new THREE.MeshPhongMaterial({
            color: Colors.brownDark,
            flatShading: true
        });

        const blade1 = new THREE.Mesh(geomBlade1, matBlade);
        blade1.position.set(8, 0, 0);
        blade1.castShadow = true;
        blade1.receiveShadow = true;

        const blade2 = new THREE.Mesh(geomBlade2, matBlade);
        blade2.position.set(8, 0, 0);
        blade2.castShadow = true;
        blade2.receiveShadow = true;
        this.propeller.add(blade1, blade2);
        this.propeller.position.set(50, 0, 0);
        this.mesh.add(this.propeller);

        // Create wheels
        const wheelProtecGeom = new THREE.BoxGeometry(30, 15, 10);
        const wheelProtecMat = new THREE.MeshPhongMaterial({
            color: Colors.white,
            flatShading: true
        });
        const wheelProtecR = new THREE.Mesh(wheelProtecGeom, wheelProtecMat);
        wheelProtecR.position.set(25, -20, 25);
        this.mesh.add(wheelProtecR);

        const wheelTireGeom = new THREE.BoxGeometry(24, 24, 4);
        const wheelTireMat = new THREE.MeshPhongMaterial({
            color: Colors.brownDark,
            flatShading: true
        });
        const wheelTireR = new THREE.Mesh(wheelTireGeom, wheelTireMat);
        wheelTireR.position.set(25, -28, 25);

        const wheelAxisGeom = new THREE.BoxGeometry(10, 10, 6);
        const wheelAxisMat = new THREE.MeshPhongMaterial({
            color: Colors.brown,
            flatShading: true
        });
        const wheelAxis = new THREE.Mesh(wheelAxisGeom, wheelAxisMat);
        wheelTireR.add(wheelAxis);

        this.mesh.add(wheelTireR);

        const wheelProtecL = wheelProtecR.clone();
        wheelProtecL.position.z = -wheelProtecR.position.z;
        this.mesh.add(wheelProtecL);

        const wheelTireL = wheelTireR.clone();
        wheelTireL.position.z = -wheelTireR.position.z;
        this.mesh.add(wheelTireL);

        const wheelTireB = wheelTireR.clone();
        wheelTireB.scale.set(.5, .5, .5);
        wheelTireB.position.set(-35, -5, 0);
        this.mesh.add(wheelTireB);

        const suspensionGeom = new THREE.BoxGeometry(4, 20, 4);
        suspensionGeom.applyMatrix4(new THREE.Matrix4().makeTranslation(0, 10, 0));
        const suspensionMat = new THREE.MeshPhongMaterial({
            color: Colors.red,
            flatShading: true
        });
        const suspension = new THREE.Mesh(suspensionGeom, suspensionMat);
        suspension.position.set(-35, -5, 0);
        suspension.rotation.z = -.3;
        this.mesh.add(suspension);

        // Initial acceleration
        this.targetX = -40;
        this.targetY = 100;
        this.t = 0;
        this.vx = 0;
        this.vy = 0;
        this.ax = (this.targetX - this.mesh.position.x) / 22500;
        this.ay = (this.targetY - this.mesh.position.y) / 22500;
    }

    nextPosition() {
        this.t++;
        if (this.t >= 300) {
            this.t = 0;
            this.targetX = normalize(randomX(), -.75, .75, -100, -20);
            this.targetY = normalize(randomY(), -.75, .75, 50, 190);
            this.vx = 0;
            this.vy = 0;
            this.ax = (this.targetX - this.mesh.position.x) / 22500;
            this.ay = (this.targetY - this.mesh.position.y) / 22500;
        }

        if (this.t > 150) {
            this.vx -= this.ax;
            this.vy -= this.ay;
        } else {
            this.vx += this.ax;
            this.vy += this.ay;
        }
        this.mesh.position.y += this.vy;
        this.mesh.position.x += this.vx;

        // Rotate plane based on velocity
        this.mesh.rotation.y = this.vx * 0.64;
        this.mesh.rotation.x = this.vy * 0.64;
        this.mesh.rotation.z = this.mesh.rotation.y;

        // Propeller rotation
        this.propeller.rotation.x += 0.3;
    }
}

// Helper functions
function randomX(): number {
    return -1 + (Math.random()) * 2;
}

function randomY(): number {
    return 1 - (Math.random()) * 2;
}

function normalize(v: number, vmin: number, vmax: number, tmin: number, tmax: number): number {
    const nv = Math.max(Math.min(v, vmax), vmin);
    const dv = vmax - vmin;
    const pc = (nv - vmin) / dv;
    const dt = tmax - tmin;
    const tv = tmin + (pc * dt);
    return tv;
}

// Create sky
function createSky(scene: THREE.Scene): Sky {
    const sky = new Sky();
    sky.mesh.position.y = -600;
    scene.add(sky.mesh);
    return sky;
}

// Create land
function createLand(scene: THREE.Scene): Land {
    const land = new Land();
    land.mesh.position.y = -600;
    scene.add(land.mesh);
    return land;
}

// Create orbit
function createOrbit(scene: THREE.Scene): Orbit {
    const orbit = new Orbit();
    orbit.mesh.position.y = -600;
    orbit.mesh.rotation.z = -Math.PI / 6;
    scene.add(orbit.mesh);
    return orbit;
}

// Create forest
function createForest(scene: THREE.Scene): Forest {
    const forest = new Forest();
    forest.mesh.position.y = -600;
    scene.add(forest.mesh);
    return forest;
}

// Create sun
function createSun(scene: THREE.Scene): Sun {
    const sun = new Sun();
    sun.mesh.scale.set(1, 1, 0.3);
    sun.mesh.position.set(0, -30, -850);
    scene.add(sun.mesh);
    return sun;
}

// Create plane
function createPlane(scene: THREE.Scene): AirPlane {
    const airplane = new AirPlane();
    airplane.mesh.scale.set(0.35, 0.35, 0.35);
    airplane.mesh.position.set(-40, 110, -250);
    scene.add(airplane.mesh);
    return airplane;
}

// Initialize the scene
export default function init(ref: HTMLDivElement) {
    const { scene, camera, renderer } = createScene(ref);
    createLights(scene);
    const airplane = createPlane(scene);
    const orbit = createOrbit(scene);
    createSun(scene);
    const land = createLand(scene);
    const forest = createForest(scene);
    const sky = createSky(scene);

    // Animation loop
    function render() {
        land.mesh.rotation.z += 0.00125;
        orbit.mesh.rotation.z += 0.00025;
        sky.mesh.rotation.z += 0.00125;
        forest.mesh.rotation.z += 0.00125;
        airplane.nextPosition();
        renderer.render(scene, camera);
        requestAnimationFrame(render);
    }

    requestAnimationFrame(render);
}