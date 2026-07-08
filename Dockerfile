###### ==============================================================================
###### ESTÁGIO DE COMPILAÇÃO (BUILD STAGE)
###### ==============================================================================
###### Utiliza a imagem oficial Go baseada em Alpine compatível com a versão do go.mod (1.21)
FROM golang:1.21-alpine AS builder

###### Instala ferramentas essenciais para compilação/dependências (git, ca-certificates, tzdata)
RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app

###### Otimização de Cache: Copia go.mod e o opcional go.sum (usando wildcard para evitar falha se go.sum não existir)
COPY go.mod go.su[m]* ./

###### Baixa as dependências especificadas nos módulos Go antes de copiar todo o código fonte
RUN go mod download

###### Copia todo o código fonte do microsserviço
COPY . .

###### Garante a geração e consistência do arquivo go.sum antes da compilação
RUN go mod tidy

###### Realiza a compilação estática do binário do Go para Linux
###### - CGO_ENABLED=0 desabilita a vinculação dinâmica com a libc do host, permitindo rodar em scratch/alpine puro
###### - GOOS=linux define o sistema operacional alvo
###### - flags "-w -s" removem tabelas de símbolos e informações de depuração, reduzindo o tamanho do binário final
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o auth-service .

###### ==============================================================================
###### ESTÁGIO DE EXECUÇÃO (FINAL STAGE)
###### ==============================================================================
###### Utiliza uma imagem Alpine recente, mínima e altamente segura
FROM alpine:3.20

###### Garante que os certificados root de autoridades certificadoras e as definições de fuso horário estão atualizados
RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

###### Criação de grupo e usuário de sistema sem privilégios (non-root)
###### Isso segue o princípio do menor privilégio para segurança em runtime
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

###### Copia apenas o binário estático compilado do estágio anterior para o ambiente final
COPY --from=builder /app/auth-service /app/auth-service

###### Ajusta a propriedade do binário e do diretório para o usuário não-root
RUN chown -R appuser:appgroup /app

###### Define a execução do container utilizando o usuário não-root criado
USER appuser

###### Define variáveis de ambiente padrão do microsserviço
ENV PORT=8001

###### Expõe a porta padrão configurada para comunicação do microsserviço
EXPOSE 8001

###### Define o binário compilado como ponto de entrada principal
ENTRYPOINT ["/app/auth-service"]
