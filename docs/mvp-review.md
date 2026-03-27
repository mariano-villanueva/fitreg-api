# FitReg — MVP Review & Demo Strategy

_Última actualización: 2026-03-19 (sesión 2)_

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
| Duplicar plantilla semanal | Hoy hay que crearlas de cero. Un coach tiene 3–4 semanas tipo que rota. |
| Campo pace/ritmo objetivo por segmento | `intensity` es string libre hoy. Un campo "5:30/km" sería más preciso para running. Workaround posible con notes. |
| ~~Historial de km del alumno~~ | ✅ **Implementado** como `TrainingLoadChart` — ver más abajo. |

### Fuera del scope del MVP (no mostrar en demo)

- Directorio de coaches y ratings → irrelevante hasta tener más de 1 coach real
- Logros → feature de perfil público, no es lo que mira el coach en la demo
- Preferencias de notificaciones → complejidad innecesaria por ahora

### Pulido necesario antes de la demo

- **Onboarding del alumno**: tiene que ser claro y rápido. Si el coach manda un link y tarda 5 minutos conectarse, se pierde.
- **Estado vacío**: la primera vez que el coach entra sin alumnos/templates, tiene que guiarlo a crear algo, no mostrar pantallas vacías.

---

## Estrategia MVP — opciones

**A) Demo mínima** — No tocar nada nuevo. Pulir lo existente, datos mock convincentes, mostrar el flujo principal. Los gaps se comunican como "en progreso".

**B) Demo + 2 features clave** ← _recomendada_ — Agregar vista semanal del alumno + dashboard de cumplimiento del coach. Son las dos features con mayor "aha moment" para alguien que hoy usa Excel.

**C) MVP real para primeros usuarios** — Lo anterior + historial de volumen + duplicar plantillas + pulido de onboarding. Suficiente para que el coach lo use con alumnos reales desde el día 1.

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

- [x] Vista semanal del alumno — WeeklyStrip en AthleteHome (puntos × 7 días, card expandida, modal inline)
- [ ] Dashboard de cumplimiento del coach (semana actual/anterior)
- [ ] Duplicar plantilla semanal
- [ ] Pulido de onboarding del alumno
- [ ] Estados vacíos con guía de primeros pasos
- [ ] Página de Pricing (planes Free/Starter/Pro/Elite, precios ARS + USD, sin tarjeta para Free)
- [ ] Datos mock para la demo
- [ ] Limpieza de DB para onboarding real en prod
