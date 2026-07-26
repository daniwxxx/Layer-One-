package model

type Profile string

const (
	ProfileExplorador   Profile = "explorador"
	ProfileOrganizador  Profile = "organizador"
	ProfileSocial       Profile = "social"
	ProfileAnalítico    Profile = "analítico"
	ProfileEstable      Profile = "estable"
	ProfileEmocional    Profile = "emocional"
	ProfileMixto        Profile = "mixto"
	ProfileIndeterminado Profile = "indeterminado"
	ProfileUnknown      Profile = "desconocido"
)

func (p Profile) Description() string {
	switch p {
	case ProfileExplorador:
		return "Persona curiosa, innovadora y abierta a nuevas experiencias."
	case ProfileOrganizador:
		return "Persona ordenada, responsable y planificadora."
	case ProfileSocial:
		return "Persona extrovertida, empática y colaborativa."
	case ProfileAnalítico:
		return "Persona reflexiva, independiente y con pensamiento abstracto."
	case ProfileEstable:
		return "Persona tranquila, equilibrada y segura."
	case ProfileEmocional:
		return "Persona sensible, ansiosa y con cambios de humor."
	case ProfileMixto:
		return "Persona con rasgos variados, sin un perfil claramente dominante."
	default:
		return "Perfil no clasificado."
	}
}

func (p Profile) CommunicationStyle() string {
	switch p {
	case ProfileExplorador:
		return "Mensajes creativos, innovadores y que desafíen lo establecido."
	case ProfileOrganizador:
		return "Mensajes estructurados, con datos y planes claros."
	case ProfileSocial:
		return "Mensajes cálidos, colaborativos y con énfasis en relaciones."
	case ProfileAnalítico:
		return "Mensajes racionales, detallados y lógicos."
	case ProfileEstable:
		return "Mensajes tranquilos, seguros y prácticos."
	case ProfileEmocional:
		return "Mensajes empáticos, comprensivos y que validen sus emociones."
	default:
		return "Mensajes equilibrados y respetuosos."
	}
}
