package dom

// TrustedHTML es marcado que el AUTOR del programa garantiza seguro. El tipo
// existe para que meter datos no confiables en el documento no compile: no
// hay conversión implícita desde string, y el único constructor obliga a
// escribir una línea que un grep encuentra.
//
// Regla: sólo literales del propio código, o el resultado de un builder de
// este ecosistema. NUNCA una cadena que venga de una petición, de una base de
// datos, de un perfil de OAuth o de otro servicio.
type TrustedHTML string

// Trust marca html como confiable. Es la ÚNICA forma de producir un
// TrustedHTML, y su nombre es lo que hace auditable el programa: buscar
// "dom.Trust(" enumera todos los puntos donde el escapado se saltea a
// propósito.
//
// Si estás por escribir dom.Trust(algoQueVinoDeAfuera), el defecto está en el
// diseño del llamador, no acá.
func Trust(html string) TrustedHTML { return TrustedHTML(html) }
