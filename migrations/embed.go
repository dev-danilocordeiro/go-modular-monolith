// Package migrations embute os arquivos SQL no binário.
//
// Cada módulo é dono das suas tabelas (prefixo do módulo no nome) e NÃO
// existem foreign keys entre módulos: a integridade entre módulos é garantida
// pela aplicação, via API pública. Isso mantém a porta aberta para extrair um
// módulo para um serviço próprio no futuro.
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
