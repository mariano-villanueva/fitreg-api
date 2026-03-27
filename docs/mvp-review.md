# FitReg — MVP Review & Demo Strategy

_Última actualización: 2026-03-27 (sesión 3)_

## Contexto

- App de running exclusivamente (expansión a otros deportes: plan a futuro)
- Target inicial: coach que hoy usa Excel, ~20–40 alumnos, sin herramienta estructurada
- Demo: mostrar con datos mock en local → si le interesa, onboarding real en prod
- Sin deadline, se itera de a poco

---

## Flujos existentes

### Coach
1. **Alta como coach** — Login con Google → solicitar ser coach (admin aprueba) → perfil público activado
2. **Gestión de alumnos** — Invitar alumno por email / aceptar solicitud del alumno → ver lista → terminar relación
3. **Crear plantilla de entreno** (daily template) — Título, tipo, distancia, segmentos (simple/intervalo) → guardar en biblioteca
4. **Crear plantilla semanal** — Nombrar semana → asignar entreno a cada día (lunes–domingo), usando plantillas o desde cero
5. **Asignar plan semanal a un alumno** — Elegir alumno, fecha de inicio (lunes), confirmar → detección de conflictos si ya tiene entrenamientos esa semana
6. **Asignar entreno suelto** — Crear assigned workout directo para un alumno con fecha, segmentos y campos esperados
7. **Ver resumen diario** — Qué alumnos tienen entreno hoy, estado (pendiente/completado/saltado)
8. **Ver detalle de entreno asignado** — Resultados cargados por el alumno, mensajes, métricas
9. **Chatear con alumno** — Mensajes dentro del contexto de un entreno asignado
10. **Gestión de logros** — Cargar resultados de carreras, toggle visibilidad pública, esperar verificación de admin
11. **Editar perfil coach** — Descripción, activar/desactivar perfil público

### Alumno
1. **Registro** — Login con Google → onboarding (datos personales)
2. **Conectarse con coach** — Solicitar coach / aceptar invitación del coach
3. **Ver entrenamientos asignados** — Lista filtrada por estado (pendiente/completado/saltado)
4. **Cargar resultado de entreno** — Marcar completado/saltado + métricas (tiempo, distancia, FC, sensación, foto)
5. **Chatear con coach** — Dentro del contexto de cada entreno
6. **Registrar entreno propio** — Log de entrenamientos personales (independiente del coach)
7. **Ver perfil de coach** — Directorio público, ratings
8. **Calificar coach** — Rating 1–5 con comentario

### Admin
1. Aprobar/rechazar solicitudes de coach
2. Ver stats generales, gestionar usuarios
3. Verificar/rechazar logros de coaches

---

## Gaps identificados

### Crítico para la demo

| Gap | Por qué importa | Esfuerzo estimado |
|-----|----------------|-------------------|
| **Vista semanal del alumno** | El alumno hoy ve una lista. Necesita ver "mi semana" como calendario lunes–domingo. Es lo primero que pregunta cualquier atleta. | Medio |
| **Resumen de volumen semanal** | El coach de running piensa en km/semana y horas/semana. No hay ningún número agregado visible hoy. | Bajo |
| **Dashboard de cumplimiento** | El coach quiere ver de un vistazo: de mis 30 alumnos, ¿quiénes completaron la semana? Hoy solo existe el resumen del día. | Medio |

### Nice to have

| Feature | Nota |
|---------|------|
| ~~Duplicar plantilla semanal~~ | ✅ **Implementado** — botón "Duplicar" en cada card, precarga el form de nueva plantilla. |
| Campo pace/ritmo objetivo por segmento | `intensity` es string libre hoy. Un campo "5:30/km" sería más preciso para running. Workaround posible con notes. |
| ~~Historial de km del alumno~~ | ✅ **Implementado** como `TrainingLoadChart` — ver más abajo. |

### Fuera del scope del MVP (no mostrar en demo)

- Directorio de coaches y ratings → irrelevante hasta tener más de 1 coach real
- Logros → feature de perfil público, no es lo que mira el coach en la demo
- Preferencias de notificaciones → complejidad innecesaria por ahora

### Pulido necesario antes de la demo

- ~~**Onboarding del alumno**~~: ✅ **Implementado** — form en 2 pasos con texto explicativo y precarga del nombre de Google.
- ~~**Estado vacío**~~: ✅ **Implementado** — panel "Primeros pasos" en dashboard + cards guiadas en todas las secciones vacías.

---

## Estrategia MVP — opciones

**A) Demo mínima** — No tocar nada nuevo. Pulir lo existente, datos mock convincentes, mostrar el flujo principal. Los gaps se comunican como "en progreso".

**B) Demo + 2 features clave** ← _recomendada_ — Agregar vista semanal del alumno + dashboard de cumplimiento del coach. Son las dos features con mayor "aha moment" para alguien que hoy usa Excel.

**C) MVP real para primeros usuarios** — Lo anterior + historial de volumen + duplicar plantillas + pulido de onboarding. Suficiente para que el coach lo use con alumnos reales desde el día 1.

---

---

## Bugs conocidos

| Bug | Descripción | Prioridad |
|-----|-------------|-----------|
| **Timezone en prod** | La semana empieza el martes cerca de la medianoche — los días se calculan en un timezone local en vez de UTC. Auditar backend: toda fecha/hora debe guardarse y calcularse en UTC. Frontend: mostrar en el timezone del usuario (usar `Intl` o `date-fns-tz`). | Alta |

---

## Deuda técnica

| Item | Descripción |
|------|-------------|
| **Unificar Workout y AssignedWorkout** | Hoy un alumno puede logear entrenamientos propios (`Workout`) y recibir entrenamientos del coach (`AssignedWorkout`). Son estructuras separadas. La idea: un entreno propio del alumno es conceptualmente un "assigned" de sí mismo. Unificar simplifca el modelo de datos, el historial y los gráficos de carga. Cambio de backend con migración. |
| **Google Cloud Storage** | Configurar GCS para almacenar fotos de resultados y futura media. Hoy no hay storage real — las fotos quedan en base64 o sin persistencia real. Requiere: crear bucket, configurar credenciales de servicio, endpoint de upload firmado. |
| **Infraestructura de emails** | Necesario para: invitaciones de coach a alumno, notificaciones de actividad, referidos. Opciones: SendGrid, Resend, AWS SES. El backend ya tiene el concepto de invitación pero no envía emails reales. |

---

## Ideas a explorar (backlog)

### Bolsa de alumnos (coaches buscan alumnos sin coach)
El flujo actual es unidireccional: el coach invita al alumno por email. Esta idea agrega el lado inverso: alumnos registrados que no tienen coach pueden activar un opt-in ("quiero que coaches me encuentren") y aparecer en un listado filtrable para coaches.
- Requiere: campo `open_to_coaches` en el perfil del alumno, endpoint de búsqueda con filtros básicos (zona, disciplina), y lógica de contacto (el coach envía solicitud, el alumno acepta).
- Es el inverso exacto del directorio de coaches que ya existe.
- **Prioridad:** baja hasta tener base de usuarios.

### Referidos
Dos variantes con motivaciones distintas:

**Alumno refiere a su coach** — el alumno comparte un link que lleva al coach a registrarse. El coach llega con credibilidad ya establecida (su alumno lo usa). El link puede pre-cargar la relación automáticamente. Es el CTA más concreto para la página de pricing: *"tu coach ya está en FitReg, unite acá"*. Canal de adquisición orgánico con alta conversión esperada.

**Coach refiere a otro coach** — menos claro el incentivo sin modelo de comisión o descuento. Podría tener sentido en comunidades de entrenadores (coach de running refiere a uno de trail). Requiere programa de beneficios para ser atractivo.

- **Prioridad:** referido de alumno → medio (alto ROI, bajo costo). Coach → baja.

### Página comercial / demo de features
Landing page aspiracional para coaches, separada de la app. Objetivo: convencer a un coach de probar FitReg antes de registrarse. Contenido tipo:
- *"Asigná semanas completas de trabajo con un clic"*
- *"Controlá la carga semanal de cada alumno en tiempo real"*
- *"Analizá el progreso y la adherencia al plan"*
- CTA: "Probá gratis con hasta 5 alumnos"

No es una página dentro de la app — es una URL pública (fitreg.app o similar) con diseño más comercial. Podría ser una SPA estática separada o una ruta pública del FE actual.
- **Prioridad:** media — necesaria antes de cualquier esfuerzo de adquisición real.

---

## Implementado en sesiones anteriores

- [x] **Carga semanal** — `TrainingLoadChart` reutilizable (coach + alumno)
  - Backend: `/coach/students/:id/load` y `/me/load` con parámetro `weeks` (4/8/12)
  - Barra única por semana: fondo gris = planificado, relleno coloreado = completado
  - Umbrales de color: rojo <50%, naranja 50–80%, verde >80%
  - Hover con tooltip: km, %, sesiones completadas/omitidas/sin marcar, flag de workouts personales
  - Etiquetas de rango semanal ("Feb 16–22", "Feb 23–Mar 1")
  - Vista coach: en `StudentWorkouts` arriba del calendario mensual
  - Vista alumno: en `AthleteHome`

---

## Modelo de negocio

- Cobrar al coach (no al alumno). El coach tiene el dolor real y la disposición a pagar.
- Planes por cantidad de alumnos activos (no por alumno variable — genera ansiedad y penaliza a los coaches con más alumnos).
- Infra actual: ~$50 USD/mes. Con 3–4 coaches en Starter se cubre.
- Precios en ARS y USD (target: coaches argentinos + internacionales).
- Pago: integración futura con MercadoPago (y posiblemente Stripe para internacionales).
- Estrategia de lanzamiento: primeros 3–5 coaches con 6 meses gratis a cambio de feedback real.

### Planes

| Plan | Límite alumnos | Notas |
|------|---------------|-------|
| **Free** | Hasta 5 | Sin tarjeta requerida (a confirmar con integración de pago) |
| **Starter** | Hasta 20 | — |
| **Pro** | Hasta 40 | — |
| **Elite** | Ilimitado | — |

_Precios en ARS/USD a definir._

---

## Plan de iteración (sin deadline)

### Bugs / infraestructura
- [ ] **Fix timezone** — auditar backend (todo en UTC), ajustar frontend al timezone del usuario
- [ ] **Google Cloud Storage** — configurar bucket, credenciales, endpoint de upload firmado
- [ ] **Infraestructura de emails** — elegir proveedor (Resend / SendGrid), integrar envío de invitaciones reales

### Features core
- [x] Vista semanal del alumno — WeeklyStrip en AthleteHome
- [x] Dashboard de cumplimiento del coach — WeeklyComplianceDashboard
- [x] Duplicar plantilla semanal
- [x] Pulido de onboarding del alumno — form en 2 pasos
- [x] Estados vacíos con guía de primeros pasos — panel + cards guiadas
- [ ] **Unificar Workout y AssignedWorkout** — entreno propio del alumno = self-assigned
- [ ] **Invitaciones reales por email** — depende de infraestructura de emails

### Adquisición / comercial
- [ ] Página comercial / demo de features (landing aspiracional para coaches)
- [ ] Página de Pricing (planes Free/Starter/Pro/Elite, precios ARS + USD)
- [ ] Referido alumno → coach ("tu coach ya está en FitReg")

### Preparación para prod
- [ ] Datos mock para la demo
- [ ] Limpieza de DB para onboarding real en prod
- [ ] Bolsa de alumnos (opt-in "quiero que coaches me encuentren") — baja prioridad
