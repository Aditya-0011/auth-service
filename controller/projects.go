package controller

import (
	"context"
	"time"

	"github.com/Aditya-0011/common/contracts/go/auth"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const projectColumns = `p.id, p.slug, p.name, p.description, p.url, p.icon, p.status, p.createdAt, p.updatedAt`

func (s *Server) queryProjects(ctx context.Context, query string, args ...any) (*auth.ProjectsResponse, error) {
	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, internal(ctx, "querying projects", err)
	}
	defer rows.Close()

	res := &auth.ProjectsResponse{}
	for rows.Next() {
		var (
			p                    auth.Project
			status               int16
			createdAt, updatedAt time.Time
		)
		if err := rows.Scan(&p.Id, &p.Slug, &p.Name, &p.Description, &p.Url, &p.Icon, &status, &createdAt, &updatedAt); err != nil {
			return nil, internal(ctx, "scanning project", err)
		}
		p.Status = auth.ProjectStatus(status)
		p.CreatedAt = timestamppb.New(createdAt)
		p.UpdatedAt = timestamppb.New(updatedAt)
		res.Projects = append(res.Projects, &p)
	}
	if err := rows.Err(); err != nil {
		return nil, internal(ctx, "iterating projects", err)
	}
	return res, nil
}

func (s *Server) ListMyProjects(ctx context.Context, req *auth.UserRequest) (*auth.ProjectsResponse, error) {
	return s.queryProjects(ctx, `
		select `+projectColumns+`
		from account.project p
		join account.creds c on c.id = $1 and c.status = 1
		where c.role = 1
		   or (p.status in (1, 2) and exists (
				select 1 from account.project_access a where a.userId = c.id and a.projectId = p.id))
		order by p.name`, req.GetUserId())
}

func (s *Server) ListProjects(ctx context.Context, req *auth.UserRequest) (*auth.ProjectsResponse, error) {
	if err := s.requireAdmin(ctx, req.GetUserId()); err != nil {
		return nil, err
	}
	return s.queryProjects(ctx, `select `+projectColumns+` from account.project p order by p.name`)
}

func (s *Server) SaveProject(ctx context.Context, req *auth.SaveProjectRequest) (*auth.SimpleResponse, error) {
	if err := s.requireAdmin(ctx, req.GetUserId()); err != nil {
		return nil, err
	}

	id := req.GetId()
	if id == 0 {
		id = -1
	}
	outcode, err := s.callOutcode(ctx, "saving project",
		`select outcode from account.edit_project($1, $2, $3, $4, $5, $6, $7)`,
		id, req.GetSlug(), req.GetName(), req.GetDescription(), req.GetUrl(), req.GetIcon(), int16(req.GetStatus()))
	if err != nil {
		return nil, err
	}

	switch outcode {
	case outSuccess:
		if req.GetId() == 0 {
			return &auth.SimpleResponse{Message: "Project created"}, nil
		}
		return &auth.SimpleResponse{Message: "Project updated"}, nil
	case outExists:
		return nil, ErrSlugTaken
	case outNotFound:
		return nil, ErrProjectNotFound
	default:
		return nil, ErrInternal
	}
}

func (s *Server) DeleteProject(ctx context.Context, req *auth.DeleteProjectRequest) (*auth.SimpleResponse, error) {
	if err := s.requireAdmin(ctx, req.GetUserId()); err != nil {
		return nil, err
	}

	outcode, err := s.callOutcode(ctx, "deleting project", `select outcode from account.delete_project($1)`, req.GetId())
	if err != nil {
		return nil, err
	}
	switch outcode {
	case outSuccess:
		return &auth.SimpleResponse{Message: "Project deleted"}, nil
	case outNotFound:
		return nil, ErrProjectNotFound
	default:
		return nil, ErrInternal
	}
}
