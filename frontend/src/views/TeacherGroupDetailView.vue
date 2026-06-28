<template>
  <div class="group-detail-container" style="max-width: 1200px; margin: 0 auto; padding: 20px; font-family: sans-serif;">
    <router-link to="/teacher/groups" style="text-decoration: none; color: #0f766e; font-weight: bold; display: inline-block; margin-bottom: 20px;">
      ⬅️ Volver a mis asignaturas
    </router-link>

    <div v-if="loading" style="text-align: center; margin-top: 40px;">
      <h3>Cargando Campus Virtual Avanzado... 🔄</h3>
    </div>

    <div v-else-if="grupo">
      <div style="background: #0f766e; color: white; padding: 20px; border-radius: 8px; margin-bottom: 25px;">
        <h2 style="margin: 0; font-size: 2rem;">📚 {{ grupo.name }}</h2>
        <p style="margin: 5px 0 0 0; opacity: 0.8; font-size: 0.9rem;">ID único del curso: {{ grupo.id }}</p>
      </div>

      <div style="display: grid; grid-template-columns: 320px 1fr; gap: 30px; align-items: start;">
        
        <div style="background: #f8fafc; border: 1px solid #e2e8f0; padding: 20px; border-radius: 8px;">
          <h3 style="margin-top: 0; color: #1e293b; border-bottom: 2px solid #e2e8f0; padding-bottom: 10px;">Matricular Alumno</h3>
          <div style="display: flex; gap: 8px; margin-bottom: 25px;">
            <input 
              v-model="nuevoEmail" 
              type="email" 
              placeholder="correo@estudiante.es"
              style="flex: 1; padding: 10px; border: 1px solid #cbd5e1; border-radius: 6px;"
              @keyup.enter="matricularAlumno"
            />
            <button @click="matricularAlumno" style="padding: 10px 14px; background-color: #10b981; color: white; border: none; border-radius: 6px; cursor: pointer; font-weight: bold;">
              ➕
            </button>
          </div>

          <h3 style="color: #1e293b;">👥 Alumnos Matriculados ({{ miembros.length }})</h3>
          <ul v-if="miembros.length > 0" style="list-style: none; padding: 0; margin: 0;">
            <li v-for="alumno in miembros" :key="alumno.id" style="padding: 12px 0; border-bottom: 1px solid #e2e8f0; display: flex; flex-direction: column; gap: 10px; font-size: 0.9rem;">
              <div style="display: flex; justify-content: space-between; align-items: center;">
                <span style="color: #334155; word-break: break-all;">📧 {{ alumno.email }}</span>
                <div style="display: flex; gap: 5px;">
                  <button @click="verEstadisticasAlumno(alumno.id)" style="background: #3b82f6; color: white; border: none; padding: 4px 8px; border-radius: 4px; cursor: pointer; font-size: 0.8rem;" title="Ver Estadísticas">
                    📊
                  </button>
                  <button @click="expulsarAlumno(alumno.id)" style="background: #ef4444; color: white; border: none; padding: 4px 8px; border-radius: 4px; cursor: pointer; font-size: 0.8rem;" title="Desmatricular Alumno">
                    🗑️
                  </button>
                </div>
              </div>

              <!-- PANEL DESPLEGABLE ULTRA SEGURO -->
              <div v-if="viendoEstadisticasDe === alumno.id" style="background: #eff6ff; padding: 15px; border-radius: 6px; border: 1px solid #bfdbfe; margin-top: 5px;">
                <h5 style="margin: 0 0 10px 0; color: #1e40af; font-size: 0.95rem;">📊 Rendimiento del Alumno</h5>
                
                <div v-if="!estadisticasAlumno">
                  <span style="color: #64748b; font-size: 0.85rem;">Cargando expediente...</span>
                </div>
                <div v-else>
                  <p style="margin: 0 0 12px 0; font-weight: bold; color: #0f172a; font-size: 0.9rem;">
                    Nota Media Global: 
                    <span :style="{ color: Number(estadisticasAlumno.total_average || 0) >= 5 ? '#166534' : '#991b1b' }">
                      {{ Number(estadisticasAlumno.total_average || 0).toFixed(2) }}/10
                    </span>
                  </p>
                  
                  <div v-if="!estadisticasAlumno.sections || estadisticasAlumno.sections.length === 0" style="color: #64748b; font-style: italic; font-size: 0.8rem;">
                    No hay temas estructurados en esta asignatura.
                  </div>
                  
                  <div v-for="sec in (estadisticasAlumno.sections || [])" :key="sec.section_id" style="margin-bottom: 6px; font-size: 0.85rem; display: flex; justify-content: space-between; align-items: center; border-bottom: 1px dashed #93c5fd; padding-bottom: 4px;">
                    <div>
                      <span style="color: #334155; display: block; font-weight: bold;">📁 {{ sec.section_title }}</span>
                      <span style="color: #64748b; font-size: 0.75rem;">{{ sec.graded_count }} trabajos evaluados</span>
                    </div>
                    <strong :style="{ color: Number(sec.average || 0) >= 5 ? '#166534' : '#991b1b', fontSize: '1rem' }">
                      {{ Number(sec.average || 0).toFixed(2) }}
                    </strong>
                  </div>
                </div>
              </div>
            </li>
          </ul>
          <div v-else style="color: #64748b; font-style: italic; font-size: 0.9rem; text-align: center; padding: 20px 0;">
            No hay alumnos asignados a este grupo.
          </div>
        </div>

        <div>
          <div style="background: #f1f5f9; border: 1px solid #cbd5e1; padding: 20px; border-radius: 8px; margin-bottom: 25px;">
            <h3 style="margin-top: 0; color: #1e293b;">📁 Crear Nueva Sección / Bloque Temático</h3>
            <div style="display: flex; gap: 10px;">
              <input v-model="nuevoTemaTitulo" type="text" placeholder="Ej: Tema 1: Arquitectura de Software" style="flex: 1; padding: 10px; border: 1px solid #cbd5e1; border-radius: 6px; background: white;" />
              <button @click="crearNuevaSeccion" style="padding: 10px 20px; background: #0f766e; color: white; border: none; border-radius: 6px; font-weight: bold; cursor: pointer;">
                Crear Sección
              </button>
            </div>
          </div>

          <h2 style="color: #1e293b; margin-bottom: 20px;">📘 Estructura de Contenidos (Moodle)</h2>
          
          <div v-if="secciones.length === 0" style="text-align: center; padding: 5px; background: #f8fafc; border: 2px dashed #cbd5e1; border-radius: 8px; color: #64748b;">
            <p>La asignatura no tiene contenido. Comienza creando una sección arriba.</p>
          </div>

          <div v-for="sec in secciones" :key="sec.id" style="background: white; border: 1px solid #cbd5e1; border-radius: 8px; margin-bottom: 25px; box-shadow: 0 4px 6px -1px rgba(0,0,0,0.05);">
            <div style="background: #f8fafc; padding: 15px 20px; border-radius: 8px 8px 0 0; border-bottom: 1px solid #cbd5e1; display: flex; justify-content: space-between; align-items: center;">
              <h3 style="margin: 0; color: #0f172a;">📁 {{ sec.title }}</h3>
              <div style="display: flex; gap: 8px;">
                <button @click="editarSeccion(sec)" style="padding: 6px 12px; background: #fbbf24; color: #78350f; border: none; border-radius: 4px; cursor: pointer; font-size: 0.85rem; font-weight: bold;">
                  ✏️ Editar Tema
                </button>
                <button @click="activarFormularioContenido(sec.id)" style="padding: 6px 12px; background: #6366f1; color: white; border: none; border-radius: 4px; cursor: pointer; font-size: 0.85rem; font-weight: bold;">
                  ➕ Añadir Contenido
                </button>
                <button @click="eliminarSeccionCompleta(sec.id)" style="padding: 6px 10px; background: #ef4444; color: white; border: none; border-radius: 4px; cursor: pointer; font-size: 0.85rem;" title="Eliminar Tema entero">
                  🗑️
                </button>
              </div>
            </div>

            <div style="padding: 20px;">
              <div v-if="seccionActivaForm === sec.id" style="background: #f8fafc; border: 1px dashed #6366f1; padding: 20px; border-radius: 6px; margin-bottom: 20px;">
                <h4 style="margin-top: 0; color: #4f46e5;">Configurar Nuevo Contenido</h4>
                
                <div style="margin-bottom: 12px;">
                  <label style="display: block; font-size: 0.85rem; font-weight: bold; margin-bottom: 4px;">Tipo de Elemento</label>
                  <select v-model="nuevoRecurso.type" style="width: 100%; padding: 8px; border: 1px solid #cbd5e1; border-radius: 4px;">
                    <option value="file">📄 Archivo / Apunte descargable (Word, PPT, PDF)</option>
                    <option value="assignment">📝 Tarea de entrega con archivo para alumnos</option>
                    <option value="quiz">❓ Cuestionario tipo test (A, B, C, D)</option>
                  </select>
                </div>

                <div style="margin-bottom: 12px;">
                  <label style="display: block; font-size: 0.85rem; font-weight: bold; margin-bottom: 4px;">Título</label>
                  <input v-model="nuevoRecurso.title" type="text" placeholder="Ej: Diapositivas Tema 1" style="width: 100%; padding: 8px; border: 1px solid #cbd5e1; border-radius: 4px; box-sizing: border-box;" />
                </div>

                <div style="margin-bottom: 12px;" v-if="nuevoRecurso.type !== 'quiz'">
                  <label style="display: block; font-size: 0.85rem; font-weight: bold; margin-bottom: 4px;">Descripción / Instrucciones</label>
                  <textarea v-model="nuevoRecurso.content" rows="2" style="width: 100%; padding: 8px; border: 1px solid #cbd5e1; border-radius: 4px; box-sizing: border-box; font-family: inherit;"></textarea>
                </div>

                <div style="margin-bottom: 12px;" v-if="nuevoRecurso.type !== 'quiz'">
                  <label style="display: block; font-size: 0.85rem; font-weight: bold; margin-bottom: 4px;">Adjuntar Archivo de Cátedra (.pdf, .docx, .pptx)</label>
                  <input type="file" @change="subirArchivoProfesor" style="width: 100%;" />
                </div>

                <div style="margin-bottom: 12px;" v-if="nuevoRecurso.type === 'assignment'">
                  <label style="display: block; font-size: 0.85rem; font-weight: bold; margin-bottom: 4px;">Fecha y Hora de Cierre (Límite)</label>
                  <input v-model="nuevoRecurso.due_at" type="datetime-local" style="width: 100%; padding: 8px; border: 1px solid #cbd5e1; border-radius: 4px; box-sizing: border-box;" />
                </div>

                <div v-if="nuevoRecurso.type === 'quiz'" style="background: white; border: 1px solid #cbd5e1; padding: 15px; border-radius: 6px; margin-bottom: 15px;">
                  <h5 style="margin: 0 0 10px 0; color: #1e293b; font-size: 1rem;">Preguntas del Test</h5>

                  <div style="background: #eef2ff; border: 1px solid #c7d2fe; padding: 12px; border-radius: 6px; margin-bottom: 14px;">
                    <h6 style="margin: 0 0 8px 0; color: #3730a3; font-size: 0.95rem;">🤖 Generar preguntas con IA</h6>
                    <div style="margin-bottom: 8px;">
                      <label style="font-size: 0.8rem; color: #475569;">Documento Word (.docx) con el material:</label>
                      <input type="file" accept=".docx" @change="iaQuiz.fileObj = $event.target.files[0]" style="display: block; margin-top: 4px; font-size: 0.85rem;" />
                    </div>
                    <textarea v-model="iaQuiz.texto" rows="3" placeholder="…o pega aquí el material en texto" style="width: 100%; padding: 6px; border: 1px solid #cbd5e1; border-radius: 4px; font-size: 0.85rem; margin-bottom: 8px; box-sizing: border-box;"></textarea>
                    <div style="display: flex; gap: 8px; flex-wrap: wrap; margin-bottom: 8px;">
                      <select v-model="iaQuiz.dificultad" style="padding: 5px; border: 1px solid #cbd5e1; border-radius: 4px; font-size: 0.85rem;">
                        <option value="baja">Dificultad baja</option>
                        <option value="media">Dificultad media</option>
                        <option value="alta">Dificultad alta</option>
                      </select>
                      <input v-model.number="iaQuiz.numPreguntas" type="number" min="1" max="30" title="Número de preguntas" style="width: 70px; padding: 5px; border: 1px solid #cbd5e1; border-radius: 4px; font-size: 0.85rem;" />
                      <input v-model="iaQuiz.enfoque" type="text" placeholder="En qué centrarse (opcional)" style="flex: 1; min-width: 140px; padding: 5px; border: 1px solid #cbd5e1; border-radius: 4px; font-size: 0.85rem;" />
                    </div>
                    <button @click="generarTestConIA" type="button" :disabled="iaQuiz.generando" style="padding: 6px 12px; background: #4f46e5; color: white; border: none; border-radius: 4px; cursor: pointer; font-size: 0.85rem; font-weight: bold;">
                      {{ iaQuiz.generando ? 'Generando…' : '✨ Generar preguntas' }}
                    </button>
                    <span v-if="iaQuiz.generando" style="margin-left: 8px; font-size: 0.8rem; color: #6366f1;">la IA está trabajando, puede tardar unos segundos…</span>
                  </div>

                  <div v-for="(p, pIdx) in nuevoRecurso.questions" :key="pIdx" style="border: 1px solid #e2e8f0; padding: 12px; border-radius: 6px; margin-bottom: 12px;">
                    <div style="display: flex; gap: 10px; margin-bottom: 8px;">
                      <input v-model="p.question_text" type="text" :placeholder="`Pregunta ${pIdx + 1}`" style="flex: 1; padding: 6px; border: 1px solid #cbd5e1; border-radius: 4px;" />
                      <button @click="eliminarPreguntaEstructura(pIdx)" style="background: #ef4444; color: white; border: none; border-radius: 4px; padding: 4px 8px; cursor: pointer;">❌</button>
                    </div>
                    <div v-for="(opt, oIdx) in p.options" :key="oIdx" style="display: flex; align-items: center; gap: 8px; margin-bottom: 4px; margin-left: 15px;">
                      <input type="radio" :name="`correct-${pIdx}`" :checked="opt.is_correct" @change="marcarCorrectaRadio(pIdx, oIdx)" />
                      <input v-model="opt.option_text" type="text" :placeholder="`Opción ${String.fromCharCode(65 + oIdx)}`" style="flex: 1; padding: 4px; border: 1px solid #e2e8f0; border-radius: 4px; font-size: 0.85rem;" />
                    </div>
                  </div>
                  <button @click="añadirPreguntaAlCuestionario" type="button" style="padding: 6px 12px; background: #e2e8f0; color: #334155; border: none; border-radius: 4px; cursor: pointer; font-size: 0.85rem; font-weight: bold;">
                    ➕ Añadir Pregunta Tipo Test
                  </button>

                  <div v-if="nuevoRecurso.questions.length > 0" style="background: #fef9c3; border: 1px solid #fde047; padding: 10px; border-radius: 6px; margin-top: 12px;">
                    <label style="font-size: 0.8rem; color: #713f12; font-weight: bold;">🪄 Mejorar las preguntas con IA</label>
                    <div style="display: flex; gap: 8px; margin-top: 6px;">
                      <input v-model="iaQuiz.instruccion" type="text" placeholder="Ej: sube la dificultad, hazlas más técnicas, mejora la pregunta 2…" style="flex: 1; padding: 5px; border: 1px solid #cbd5e1; border-radius: 4px; font-size: 0.85rem;" />
                      <button @click="mejorarTestConIA" type="button" :disabled="iaQuiz.mejorando" style="padding: 6px 12px; background: #ca8a04; color: white; border: none; border-radius: 4px; cursor: pointer; font-size: 0.85rem; font-weight: bold; white-space: nowrap;">
                        {{ iaQuiz.mejorando ? 'Mejorando…' : 'Mejorar' }}
                      </button>
                    </div>
                  </div>
                </div>

                <div style="display: flex; justify-content: flex-end; gap: 10px;">
                  <button @click="seccionActivaForm = null" style="padding: 8px 16px; background: white; border: 1px solid #cbd5e1; border-radius: 4px; cursor: pointer;">Cancelar</button>
                  <button @click="guardarContenidoEnServidor(sec.id)" style="padding: 8px 16px; background: #10b981; color: white; border: none; border-radius: 4px; font-weight: bold; cursor: pointer;">Save Elemento</button>
                </div>
              </div>

              <ul v-if="sec.resources && sec.resources.length > 0" style="list-style: none; padding: 0; margin: 0;">
                <li v-for="res in sec.resources" :key="res.id" style="padding: 15px; border: 1px solid #e2e8f0; border-radius: 6px; margin-bottom: 12px; background: #f8fafc;">
                  <div style="display: flex; justify-content: space-between; align-items: flex-start;">
                    <div>
                      <span style="font-size: 1.3rem; margin-right: 8px;">
                        {{ res.type === 'file' ? '📄' : res.type === 'assignment' ? '📝' : '❓' }}
                      </span>
                      <strong style="color: #1e293b; font-size: 1.1rem;">{{ res.title }}</strong>
                      <p style="margin: 6px 0 0 28px; color: #475569; font-size: 0.95rem;">{{ res.content }}</p>
                      
                      <div v-if="res.file_path" style="margin: 6px 0 0 28px;">
                        <button @click="descargarArchivoSeguro(res.file_path, res.title)" style="background: none; border: none; color: #2563eb; font-size: 0.85rem; font-weight: bold; cursor: pointer; text-decoration: underline; padding: 0;">
                          📥 Descargar Archivo Cátedra
                        </button>
                      </div>

                      <div v-if="res.due_at" style="margin: 6px 0 0 28px; font-size: 0.85rem; color: #dc2626; font-weight: bold;">
                        ⏰ Límite: {{ new Date(res.due_at).toLocaleString() }}
                      </div>
                    </div>
                    
                    <div style="display: flex; gap: 8px;">
                      <button @click="editarRecurso(res)" style="padding: 5px 10px; background: #fbbf24; color: #78350f; border: none; border-radius: 4px; font-size: 0.8rem; cursor: pointer; font-weight: bold;">
                        ✏️ Editar
                      </button>
                      <button v-if="res.type === 'assignment' || res.type === 'quiz'" @click="revisarEntregasAlumnos(res.id)" style="padding: 5px 10px; background: #0f766e; color: white; border: none; border-radius: 4px; font-size: 0.8rem; cursor: pointer; font-weight: bold;">
                        📊 Ver Entregas
                      </button>
                      <button @click="eliminarRecursoServidor(res.id)" style="background: #ef4444; color: white; padding: 5px 10px; border: none; border-radius: 4px; font-size: 0.8rem; cursor: pointer;">
                        🗑️
                      </button>
                    </div>
                  </div>

                  <div v-if="revisandoRecursoId === res.id" style="margin-top: 15px; background: white; border: 1px solid #cbd5e1; padding: 15px; border-radius: 6px;">
                    <h5 style="margin: 0 0 12px 0; color: #0f766e; font-size: 1rem; border-bottom: 1px dashed #cbd5e1; padding-bottom: 4px;">Corrección de Alumnos</h5>
                    
                    <p v-if="entregasAlumnos.length === 0" style="color: #64748b; font-style: italic; font-size: 0.9rem; margin: 0;">Nadie ha enviado la tarea todavía.</p>
                    
                    <div v-for="sub in entregasAlumnos" :key="sub.id" style="border-bottom: 1px solid #f1f5f9; padding: 10px 0; display: flex; flex-direction: column; gap: 6px;">
                      <div style="display: flex; justify-content: space-between; font-size: 0.9rem;">
                        <span style="font-weight: bold; color: #334155;">👨‍🎓 {{ sub.student_email }}</span>
                        <span style="color: #64748b; font-size: 0.8rem;">Enviado: {{ new Date(sub.submitted_at).toLocaleString() }}</span>
                      </div>
                      <p style="margin: 0; padding: 6px; background: #f8fafc; border-radius: 4px; font-size: 0.9rem; color: #475569;">{{ sub.text_content || 'Sin texto explicativo.' }}</p>
                      
                      <div v-if="sub.file_path" style="margin-top: 5px;">
                        <button @click="descargarArchivoSeguro(sub.file_path, `Entrega_${sub.student_email}`)" style="background: none; border: none; color: #2563eb; font-size: 0.85rem; font-weight: bold; cursor: pointer; text-decoration: underline; padding: 0;">
                          📥 Descargar Trabajo del Alumno
                        </button>
                      </div>

                      <div style="display: flex; gap: 10px; align-items: center; margin-top: 6px; background: #f8fafc; padding: 8px; border-radius: 4px;">
                        <input v-model="notasFormulario[sub.student_id]" type="number" placeholder="Nota" step="0.1" max="10" style="width: 70px; padding: 4px; border: 1px solid #cbd5e1; border-radius: 4px;" />
                        <input v-model="feedbackFormulario[sub.student_id]" type="text" placeholder="Feedback del profesor..." style="flex: 1; padding: 4px; border: 1px solid #cbd5e1; border-radius: 4px;" />
                        <button @click="enviarCalificacion(res.id, sub.student_id)" style="padding: 4px 10px; background: #10b981; color: white; border: none; border-radius: 4px; cursor: pointer; font-weight: bold; font-size: 0.8rem;">
                          Calificar
                        </button>
                        <span v-if="sub.grade !== undefined && sub.grade !== null" style="font-size: 0.85rem; background: #dcfce3; color: #166534; padding: 2px 6px; border-radius: 4px; font-weight: bold;">
                          Nota actual: {{ sub.grade }}/10
                        </span>
                      </div>

                      <div style="margin-top: 8px;" v-if="res.type === 'quiz'">
                        <button @click="verExamenDetalladoAlumno(res.id, sub.student_id)" style="padding: 4px 10px; background: #6366f1; color: white; border: none; border-radius: 4px; cursor: pointer; font-size: 0.8rem; font-weight: bold;">
                          🔍 Inspeccionar Respuestas Elegidas por el Alumno
                        </button>
                      </div>

                      <div v-if="examenInspeccionado && examenInspeccionadoStudentId === sub.student_id && revisandoRecursoId === res.id" style="margin-top: 10px; background: #f8fafc; border: 1px solid #cbd5e1; padding: 12px; border-radius: 6px;">
                        <h6 style="margin: 0 0 10px 0; color: #4f46e5; font-size: 0.9rem;">Hojas de Respuestas de este Estudiante:</h6>
                        <div v-for="q in examenInspeccionado.questions" :key="q.id" style="margin-bottom: 8px; font-size: 0.85rem;">
                          <strong>❓ {{ q.question_text }}</strong>
                          <div v-for="o in q.options" :key="o.id" style="margin-left: 10px; color: #334155;" :style="o.selected ? {fontWeight: 'bold', color: o.is_correct ? 'green' : 'red'} : o.is_correct ? {color: 'green'} : {}">
                            • {{ o.option_text }} 
                            <span v-if="o.selected">[Marcada por Alumno]</span>
                            <span v-if="o.is_correct">✓</span>
                          </div>
                        </div>
                      </div>

                    </div>
                  </div>

                </li>
              </ul>
              <p v-else style="color: #94a3b8; font-style: italic; margin: 0;">Tema vacío.</p>
            </div>
          </div>
        </div>

      </div>
    </div>
  </div>
</template>

<script setup>
import { API_BASE } from '@/services/apiBase'
import { ref, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const route = useRoute()
const auth = useAuthStore()
const grupoId = route.params.id

const grupo = ref(null)
const miembros = ref([])
const secciones = ref([])
const loading = ref(true)

const nuevoEmail = ref('')
const nuevoTemaTitulo = ref('')
const seccionActivaForm = ref(null)

const viendoEstadisticasDe = ref(null)
const estadisticasAlumno = ref(null)

const objetoRecursoVacio = () => ({
  type: 'file',
  title: '',
  content: '',
  due_at: '',
  fileObj: null,
  questions: []
})
const nuevoRecurso = ref(objetoRecursoVacio())

// Estado del generador/mejorador de tests con IA.
const iaQuiz = ref({
  generando: false,
  mejorando: false,
  texto: '',
  fileObj: null,
  dificultad: 'media',
  numPreguntas: 5,
  enfoque: '',
  instruccion: ''
})

const revisandoRecursoId = ref(null)
const entregasAlumnos = ref([])
const notasFormulario = ref({})
const feedbackFormulario = ref({})

const examenInspeccionado = ref(null)
const examenInspeccionadoStudentId = ref(null)

function getHeaders(isMultipart = false) {
  const h = { 'Authorization': `Bearer ${auth.token}` }
  if (!isMultipart) h['Content-Type'] = 'application/json'
  return h
}

const cargarLmsCompleto = async () => {
  try {
    loading.value = true
    const resG = await fetch(`${API_BASE}/api/groups/${grupoId}`, { headers: getHeaders() })
    if (resG.ok) {
      const d = await resG.json()
      grupo.value = d
      miembros.value = d.members || []
    }
    const resC = await fetch(`${API_BASE}/api/groups/${grupoId}/content`, { headers: getHeaders() })
    if (resC.ok) secciones.value = await resC.json() || []
  } catch (e) { console.error(e) }
  finally { loading.value = false }
}

const matricularAlumno = async () => {
  if (!nuevoEmail.value.trim()) return
  try {
    const res = await fetch(`${API_BASE}/api/groups/${grupoId}/members`, {
      method: 'POST',
      headers: getHeaders(),
      body: JSON.stringify({ emails: [nuevoEmail.value.trim()] })
    })
    if (res.ok) {
      nuevoEmail.value = ''
      const data = await res.json()
      miembros.value = data || []
      alert("Estudiante matriculado con éxito 🚀")
    }
  } catch (e) { console.error(e) }
}

const expulsarAlumno = async (memberId) => {
  if (!confirm("¿Seguro que quieres desmatricular a este alumno de la asignatura?")) return
  try {
    const res = await fetch(`${API_BASE}/api/groups/${grupoId}/members/${memberId}`, {
      method: 'DELETE',
      headers: getHeaders()
    })
    if (res.ok) {
      miembros.value = miembros.value.filter(m => m.id !== memberId)
      alert("Alumno eliminado de la asignatura.")
    }
  } catch (e) { console.error(e) }
}

const crearNuevaSeccion = async () => {
  if (!nuevoTemaTitulo.value.trim()) return
  try {
    const res = await fetch(`${API_BASE}/api/groups/${grupoId}/sections`, {
      method: 'POST',
      headers: getHeaders(),
      body: JSON.stringify({ title: nuevoTemaTitulo.value.trim(), position: secciones.value.length })
    })
    if (res.ok) {
      nuevoTemaTitulo.value = ''
      cargarLmsCompleto()
    }
  } catch (e) { console.error(e) }
}

const eliminarSeccionCompleta = async (sectionId) => {
  if (!confirm("¿Estás seguro? Se borrará el tema y todos los archivos y cuestionarios de su interior.")) return
  try {
    const res = await fetch(`${API_BASE}/api/sections/${sectionId}`, { method: 'DELETE', headers: getHeaders() })
    if (res.ok) cargarLmsCompleto()
  } catch (e) { console.error(e) }
}

const editarSeccion = async (sec) => {
  const nuevoTitulo = prompt("Modificar título del tema:", sec.title)
  if (!nuevoTitulo || nuevoTitulo === sec.title) return
  try {
    const res = await fetch(`${API_BASE}/api/sections/${sec.id}`, {
      method: 'PUT',
      headers: getHeaders(),
      body: JSON.stringify({ title: nuevoTitulo })
    })
    if (res.ok) cargarLmsCompleto()
  } catch(e) { console.error(e) }
}

const editarRecurso = async (res) => {
  const nuevoTitulo = prompt("Modificar título:", res.title)
  if (!nuevoTitulo) return
  const nuevaDesc = prompt("Modificar descripción:", res.content) || ""
  
  let due_at = res.due_at
  if (res.type === 'assignment') {
    const fecha = prompt("Modificar Fecha límite (YYYY-MM-DD) o vacío:", res.due_at ? res.due_at.split('T')[0] : "")
    due_at = fecha ? new Date(fecha).toISOString() : null
  }

  try {
    const fetchRes = await fetch(`${API_BASE}/api/resources/${res.id}`, {
      method: 'PUT',
      headers: getHeaders(),
      body: JSON.stringify({ title: nuevoTitulo, content: nuevaDesc, due_at })
    })
    if (fetchRes.ok) cargarLmsCompleto()
  } catch(e) { console.error(e) }
}

const activarFormularioContenido = (secId) => {
  seccionActivaForm.value = secId
  nuevoRecurso.value = objetoRecursoVacio()
}

const subirArchivoProfesor = (e) => {
  if (e.target.files && e.target.files.length > 0) {
    nuevoRecurso.value.fileObj = e.target.files[0]
  }
}

const añadirPreguntaAlCuestionario = () => {
  nuevoRecurso.value.questions.push({
    question_text: '',
    options: [
      { option_text: '', is_correct: true },
      { option_text: '', is_correct: false },
      { option_text: '', is_correct: false },
      { option_text: '', is_correct: false }
    ]
  })
}

const marcarCorrectaRadio = (pIdx, oIdx) => {
  nuevoRecurso.value.questions[pIdx].options.forEach((opt, idx) => {
    opt.is_correct = (idx === oIdx)
  })
}

const eliminarPreguntaEstructura = (idx) => {
  nuevoRecurso.value.questions.splice(idx, 1)
}

// Genera preguntas con IA a partir de un .docx y/o texto. No guarda nada: rellena
// el editor de preguntas para que el profesor las revise/edite antes de guardar.
const generarTestConIA = async () => {
  if (!iaQuiz.value.texto.trim() && !iaQuiz.value.fileObj) {
    return alert('Sube un .docx o pega el material en texto.')
  }
  iaQuiz.value.generando = true
  try {
    const fd = new FormData()
    if (iaQuiz.value.fileObj) fd.append('file', iaQuiz.value.fileObj)
    if (iaQuiz.value.texto.trim()) fd.append('text', iaQuiz.value.texto)
    fd.append('difficulty', iaQuiz.value.dificultad)
    fd.append('num_questions', String(iaQuiz.value.numPreguntas))
    fd.append('focus', iaQuiz.value.enfoque)

    const res = await fetch(`${API_BASE}/api/quizzes/ai-generate`, {
      method: 'POST',
      headers: getHeaders(true),
      body: fd
    })
    const data = await res.json()
    if (!res.ok) throw new Error(data.error || 'No se pudo generar el test')
    nuevoRecurso.value.questions = data.questions || []
    if (!nuevoRecurso.value.title.trim()) nuevoRecurso.value.title = 'Test generado con IA'
  } catch (e) {
    alert('IA: ' + e.message)
  } finally {
    iaQuiz.value.generando = false
  }
}

// Mejora las preguntas actuales (subir dificultad, nivel, reformular...) según la
// instrucción del profesor. Reemplaza las preguntas del editor con la versión mejorada.
const mejorarTestConIA = async () => {
  if (nuevoRecurso.value.questions.length === 0) {
    return alert('Primero genera o añade preguntas.')
  }
  iaQuiz.value.mejorando = true
  try {
    const res = await fetch(`${API_BASE}/api/quizzes/ai-improve`, {
      method: 'POST',
      headers: getHeaders(),
      body: JSON.stringify({
        instruction: iaQuiz.value.instruccion,
        questions: nuevoRecurso.value.questions
      })
    })
    const data = await res.json()
    if (!res.ok) throw new Error(data.error || 'No se pudo mejorar el test')
    nuevoRecurso.value.questions = data.questions || []
  } catch (e) {
    alert('IA: ' + e.message)
  } finally {
    iaQuiz.value.mejorando = false
  }
}

const guardarContenidoEnServidor = async (secId) => {
  if (!nuevoRecurso.value.title.trim()) return alert("El título es obligatorio")

  try {
    if (nuevoRecurso.value.type === 'quiz') {
      const res = await fetch(`${API_BASE}/api/sections/${secId}/quizzes`, {
        method: 'POST',
        headers: getHeaders(),
        body: JSON.stringify({ title: nuevoRecurso.value.title, questions: nuevoRecurso.value.questions })
      })
      if (res.ok) {
        seccionActivaForm.value = null
        cargarLmsCompleto()
      }
    } else {
      const fd = new FormData()
      fd.append("type", nuevoRecurso.value.type)
      fd.append("title", nuevoRecurso.value.title)
      fd.append("content", nuevoRecurso.value.content)
      if (nuevoRecurso.value.due_at) fd.append("due_at", new Date(nuevoRecurso.value.due_at).toISOString())
      if (nuevoRecurso.value.fileObj) fd.append("file", nuevoRecurso.value.fileObj)

      const res = await fetch(`${API_BASE}/api/sections/${secId}/resources`, {
        method: 'POST',
        headers: getHeaders(true),
        body: fd
      })
      if (res.ok) {
        seccionActivaForm.value = null
        cargarLmsCompleto()
      }
    }
  } catch (e) { console.error(e) }
}

const eliminarRecursoServidor = async (resId) => {
  if (!confirm("¿Borrar este elemento de la asignatura?")) return
  try {
    const res = await fetch(`${API_BASE}/api/resources/${resId}`, { method: 'DELETE', headers: getHeaders() })
    if (res.ok) cargarLmsCompleto()
  } catch (e) { console.error(e) }
}

const revisarEntregasAlumnos = async (resId) => {
  if (revisandoRecursoId.value === resId) {
    revisandoRecursoId.value = null
    return
  }
  try {
    const res = await fetch(`${API_BASE}/api/resources/${resId}/submissions`, { headers: getHeaders() })
    if (res.ok) {
      entregasAlumnos.value = await res.json() || []
      revisandoRecursoId.value = resId
      entregasAlumnos.value.forEach(sub => {
        if (sub.grade !== undefined && sub.grade !== null) notasFormulario.value[sub.student_id] = sub.grade
        if (sub.feedback) feedbackFormulario.value[sub.student_id] = sub.feedback
      })
    }
  } catch (e) { console.error(e) }
}

const enviarCalificacion = async (resId, studentId) => {
  const nota = notasFormulario.value[studentId]
  const feedback = feedbackFormulario.value[studentId] || ""
  if (nota === undefined || nota === "") return alert("Pon una nota válida.")

  try {
    const res = await fetch(`${API_BASE}/api/resources/${resId}/submissions/${studentId}/grade`, {
      method: 'POST',
      headers: getHeaders(),
      body: JSON.stringify({ grade: String(nota), feedback })
    })
    if (res.ok) {
      alert("¡Nota guardada e incorporada al expediente del alumno! 🎓")
      revisandoRecursoId.value = null
      cargarLmsCompleto()
    }
  } catch (e) { console.error(e) }
}

const verExamenDetalladoAlumno = async (resourceId, studentId) => {
  if (examenInspeccionadoStudentId.value === studentId) {
    examenInspeccionado.value = null
    examenInspeccionadoStudentId.value = null
    return
  }
  try {
    const res = await fetch(`${API_BASE}/api/resources/${resourceId}/review/${studentId}`, { headers: getHeaders() })
    if (res.ok) {
      examenInspeccionado.value = await res.json()
      examenInspeccionadoStudentId.value = studentId
    }
  } catch (e) { console.error(e) }
}

const descargarArchivoSeguro = async (filePath, title) => {
  try {
    const res = await fetch(`${API_BASE}/api/uploads/${filePath}`, {
      headers: { 'Authorization': `Bearer ${auth.token}` }
    })
    if (!res.ok) throw new Error("Fallo en la descarga")
    const blob = await res.blob()
    const url = window.URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    const ext = filePath.includes('.') ? '.' + filePath.split('.').pop() : ''
    a.download = `${title}${ext}`
    a.click()
    window.URL.revokeObjectURL(url)
  } catch (error) {
    alert("No se pudo descargar el archivo.")
    console.error(error)
  }
}

// FUNCIÓN DE ESTADÍSTICAS REPARADA COMPLEMENTADA CON PARSEO DEFENSIVO
const verEstadisticasAlumno = async (studentId) => {
  if (viendoEstadisticasDe.value === studentId) {
    viendoEstadisticasDe.value = null
    return
  }
  try {
    const res = await fetch(`${API_BASE}/api/groups/${grupoId}/students/${studentId}/stats`, { headers: getHeaders() })
    if (res.ok) {
      const data = await res.json()
      
      // Si la API de Go envía las secciones nulas o ausentes, forzamos un array vacío reactivo
      if (!data || !data.sections) {
        if (data) data.sections = []
      }
      
      estadisticasAlumno.value = data
      viendoEstadisticasDe.value = studentId
    } else {
      estadisticasAlumno.value = { total_average: 0, sections: [] }
      viendoEstadisticasDe.value = studentId
    }
  } catch (e) { 
    console.error(e)
    estadisticasAlumno.value = { total_average: 0, sections: [] }
    viendoEstadisticasDe.value = studentId
  }
}

onMounted(() => { cargarLmsCompleto() })
</script>